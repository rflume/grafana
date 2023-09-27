package serviceauthreg

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/grafana/pkg/components/satokengen"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/slugify"
	"github.com/grafana/grafana/pkg/models/roletype"
	ac "github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/apikey"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/oauthserver"
	"github.com/grafana/grafana/pkg/services/secrets"
	"github.com/grafana/grafana/pkg/services/secrets/kvstore"
	"github.com/grafana/grafana/pkg/services/serviceaccounts"
	"github.com/grafana/grafana/pkg/services/serviceauth"
)

// TODO (gamab) add server lock

var _ serviceauth.ExternalServiceRegistry = &Registry{}

const (
	tmpOrgID = 1 // TODO (gamab) fix OrgID
	skvType  = "external-service-token"
)

type Registry struct {
	acSvc       ac.Service
	features    featuremgmt.FeatureToggles
	logger      log.Logger
	oauthServer oauthserver.OAuth2Server
	saSvc       serviceaccounts.Service
	skvStore    kvstore.SecretsKVStore
}

func ProvideServiceAuthRegistry(acSvc ac.Service, saSvc serviceaccounts.Service, oauthServer oauthserver.OAuth2Server,
	sqlStore db.DB, secretSvc secrets.Service, features featuremgmt.FeatureToggles) *Registry {
	return &Registry{
		acSvc:       acSvc,
		features:    features,
		logger:      log.New("serviceauth.registry"),
		oauthServer: oauthServer,
		saSvc:       saSvc,
		skvStore:    kvstore.NewSQLSecretsKVStore(sqlStore, secretSvc, log.New("serviceauth.registry.skv")), // TODO (gamab) Not sure I'm allowed to use this but it prevents a cyclic dependency
	}
}

// SaveExternalService implements serviceauth.ExternalServiceRegistry.
func (r *Registry) SaveExternalService(ctx context.Context, cmd *serviceauth.ExternalServiceRegistration) (*serviceauth.ExternalServiceDTO, error) {
	switch cmd.AuthProvider {
	case serviceauth.OAuth2Server:
		if !r.features.IsEnabled(featuremgmt.FlagExternalServiceAuth) {
			r.logger.Warn("Skipping external service authentication, flag disabled", "service", cmd.Name, "flag", featuremgmt.FlagExternalServiceAuth)
			return nil, nil
		}
		r.logger.Debug("Routing the External Service registration to the OAuth2Server", "service", cmd.Name)
		return r.oauthServer.SaveExternalService(ctx, cmd)
	case serviceauth.ServiceAccounts:
		if !r.features.IsEnabled(featuremgmt.FlagExternalServiceAccounts) {
			r.logger.Warn("Skipping external service authentication, flag disabled", "service", cmd.Name, "flag", featuremgmt.FlagExternalServiceAccounts)
			return nil, nil
		}
		r.logger.Debug("Handling the External Service registration", "service", cmd.Name)
		return r.SaveSATokenExternalService(ctx, cmd)
	default:
		return nil, serviceauth.ErrUnknownProvider.Errorf("unknow provider '%v'", cmd.AuthProvider)
	}
}

func (r *Registry) SaveSATokenExternalService(ctx context.Context, cmd *serviceauth.ExternalServiceRegistration) (*serviceauth.ExternalServiceDTO, error) {
	slug := slugify.Slugify(cmd.Name)

	if cmd.Impersonation.Enabled {
		r.logger.Warn("Impersonation is not handled when using service account token", "service", slug)
	}

	saID, errRetrieve := r.saSvc.RetrieveServiceAccountIdByName(ctx, tmpOrgID, slug)
	if errRetrieve != nil && !errors.Is(errRetrieve, serviceaccounts.ErrServiceAccountNotFound) {
		return nil, errRetrieve
	}

	if !cmd.Self.Enabled || len(cmd.Self.Permissions) == 0 {
		if saID > 0 {
			r.logger.Info("Self disabled. Deleting previous service account", "service", slug, "permission count", len(cmd.Self.Permissions), "serviceaccount", saID)
			r.deleteServiceAccount(ctx, slug, saID)
		}
		r.logger.Info("Self disabled. Skipping service account creation", "service", slug, "permission count", len(cmd.Self.Permissions))
		return nil, nil
	}
	saID, token, errSave := r.saveServiceAccount(ctx, slug, saID, cmd.Self.Permissions)
	if errSave != nil {
		r.logger.Error("Could not save service account", "service", slug, "error", errSave.Error())
		return nil, errSave
	}

	return &serviceauth.ExternalServiceDTO{
		Name:         cmd.Name,
		ID:           slug,
		Secret:       token,
		AuthProvider: serviceauth.ServiceAccounts,
		Extra:        nil,
	}, nil
}

// saveServiceAccount creates or update the service account associated with this external service
func (r *Registry) saveServiceAccount(ctx context.Context, slug string, saID int64, permissions []ac.Permission) (int64, string, error) {
	if saID <= 0 {
		// Create a service account
		r.logger.Debug("Create service account", "service", slug)
		return r.createServiceAccount(ctx, slug, permissions)
	}

	// update the service account's permissions
	r.logger.Debug("Update role permissions", "service", slug, "saID", saID)
	if err := r.acSvc.SaveExternalServiceRole(ctx, ac.SaveExternalServiceRoleCommand{
		OrgID:             ac.GlobalOrgID,
		Global:            true,
		ExternalServiceID: slug,
		ServiceAccountID:  saID,
		Permissions:       permissions,
	}); err != nil {
		return 0, "", err
	}

	// FIXME (gamab) make sure skv token and api key store are in phase
	r.logger.Debug("Get service account token from skv", "service", slug, "saID", saID)
	token, ok, err := r.skvStore.Get(ctx, tmpOrgID, slug, skvType)
	if err != nil {
		return 0, "", err
	}
	if !ok {
		token, err := r.createServiceAccountToken(ctx, slug, saID)
		return saID, token.Key, err
	}

	return saID, token, nil
}

// deleteServiceAccount deletes a service account by ID and removes its associated role
func (r *Registry) deleteServiceAccount(ctx context.Context, slug string, saID int64) error {
	r.logger.Debug("Delete service account", "service", slug, "saID", saID)
	if err := r.saSvc.DeleteServiceAccount(ctx, tmpOrgID, saID); err != nil {
		return err
	}
	return r.acSvc.DeleteExternalServiceRole(ctx, slug)
}

// createServiceAccount creates a service account with the given permissions and returns the ID of the service account
// When no permission is given, the account isn't created and NoServiceAccountID is returned
// This first design does not use a single transaction for the whole service account creation process => database consistency is not guaranteed.
// Consider changing this in the future.
func (r *Registry) createServiceAccount(ctx context.Context, slug string, permissions []ac.Permission) (int64, string, error) {
	if len(permissions) == 0 {
		// No permission, no service account
		r.logger.Debug("No permission, no service account", "service", slug)
		return 0, "", nil
	}

	newRole := func(r roletype.RoleType) *roletype.RoleType {
		return &r
	}
	newBool := func(b bool) *bool {
		return &b
	}

	r.logger.Debug("Generate service account", "service", slug, "orgID", tmpOrgID)
	sa, err := r.saSvc.CreateServiceAccount(ctx, tmpOrgID, &serviceaccounts.CreateServiceAccountForm{
		Name:       slug,
		Role:       newRole(roletype.RoleNone),
		IsDisabled: newBool(false),
	})
	if err != nil {
		return 0, "", err
	}

	token, err := r.createServiceAccountToken(ctx, slug, sa.Id)
	if err != nil {
		return 0, "", err
	}

	r.logger.Debug("Create tailored role for service account", "service", slug, "service_account_id", sa.Id, "permissions", fmt.Sprintf("%v", permissions))
	if err := r.acSvc.SaveExternalServiceRole(ctx, ac.SaveExternalServiceRoleCommand{
		OrgID:             ac.GlobalOrgID,
		Global:            true,
		ExternalServiceID: slug,
		ServiceAccountID:  sa.Id,
		Permissions:       permissions,
	}); err != nil {
		return 0, "", err
	}

	return sa.Id, token.Key, nil
}

func (r *Registry) createServiceAccountToken(ctx context.Context, slug string, saID int64) (*apikey.APIKey, error) {
	r.logger.Debug("Generate new key", "service", slug, "orgID", tmpOrgID)
	newKeyInfo, err := satokengen.New(slug)
	if err != nil {
		return nil, err
	}

	r.logger.Debug("Generate service account token", "service", slug, "orgID", tmpOrgID)
	token, err := r.saSvc.AddServiceAccountToken(ctx, saID, &serviceaccounts.AddServiceAccountTokenCommand{
		Name:  "token-" + slug,
		OrgId: tmpOrgID,
		Key:   newKeyInfo.HashedKey,
	})
	if err != nil {
		return nil, err
	}

	r.logger.Debug("Save service account token in skv", "service", slug, "orgID", tmpOrgID)
	if err = r.skvStore.Set(ctx, tmpOrgID, slug, skvType, newKeyInfo.ClientSecret); err != nil {
		return nil, err
	}
	return token, nil
}