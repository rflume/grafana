package provisioning

import (
	ac "github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/ngalert/accesscontrol"
)

var RuleActions = accesscontrol.ActionsProvider{
	Create: ac.ActionAlertingProvisioningRuleCreate,
	Read:   ac.ActionAlertingProvisioningRuleRead,
	Update: ac.ActionAlertingProvisioningRuleUpdate,
	Delete: ac.ActionAlertingProvisioningRuleDelete,
}
