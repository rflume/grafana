package serviceaccounts

import (
	"context"

	"github.com/grafana/grafana/pkg/services/apikey"
)

/*
ServiceAccountService is the service that manages service accounts.

Service accounts are used to authenticate API requests. They are not users and
do not have a password.
*/
type Service interface {
	AddServiceAccountToken(ctx context.Context, serviceAccountID int64,
		cmd *AddServiceAccountTokenCommand) (*apikey.APIKey, error)
	CreateServiceAccount(ctx context.Context, orgID int64, saForm *CreateServiceAccountForm) (*ServiceAccountDTO, error)
	DeleteServiceAccount(ctx context.Context, orgID, serviceAccountID int64) error
	RetrieveServiceAccount(ctx context.Context, orgID, serviceAccountID int64) (*ServiceAccountProfileDTO, error)
	RetrieveServiceAccountIdByName(ctx context.Context, orgID int64, name string) (int64, error)
	UpdateServiceAccount(ctx context.Context, orgID, serviceAccountID int64,
		saForm *UpdateServiceAccountForm) (*ServiceAccountProfileDTO, error)

	// functions needed for serviceaccounts/api/api.go

	ListTokens(ctx context.Context, query *GetSATokensQuery) ([]apikey.APIKey, error)
	MigrateApiKey(ctx context.Context, orgID int64, keyId int64) error
	MigrateApiKeysToServiceAccounts(ctx context.Context, orgID int64) (*MigrationResult, error)
	SearchOrgServiceAccounts(ctx context.Context, query *SearchOrgServiceAccountsQuery) (*SearchOrgServiceAccountsResult, error)

	// Service account tokens
	DeleteServiceAccountToken(ctx context.Context, orgID, serviceAccountID, tokenID int64) error
}
