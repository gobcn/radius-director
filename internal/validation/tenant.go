package validation

import (
	"fmt"

	"github.com/gobcn/radius-director/internal/model"
	"github.com/gobcn/radius-director/internal/templates"
)

func validateTenants(tenants map[string]model.Tenant, templateLoader templates.Loader) []error {
	var validationErrors []error
	for _, identifier := range sortedKeys(tenants) {
		validationErrors = append(validationErrors, validateTenant(identifier, tenants[identifier], templateLoader)...)
	}

	return validationErrors
}

func validateTenant(identifier string, tenant model.Tenant, templateLoader templates.Loader) []error {
	var validationErrors []error
	if tenant.AuthenticationProfile == "" {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: authentication_profile must be specified", identifier))
	}
	if tenant.DeploymentProfile == "" {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: deployment_profile must be specified", identifier))
	}
	if tenant.Database == (model.Database{}) {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: exactly one database must be defined", identifier))
	} else {
		validationErrors = append(validationErrors, validateDatabase(identifier, tenant.Database)...)
	}
	if tenant.RADIUSServer == (model.RADIUSServer{}) {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: exactly one radius server must be defined", identifier))
	} else {
		validationErrors = append(validationErrors, validateRADIUSServer(identifier, tenant.RADIUSServer, templateLoader)...)
	}
	if len(tenant.NASAssignments) == 0 {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: at least one nas assignment must be defined", identifier))
	} else {
		validationErrors = append(validationErrors, validateNASAssignments(identifier, tenant.NASAssignments)...)
	}
	validationErrors = append(validationErrors, validateTrustedRADIUSClientAssignments(identifier, tenant.TrustedRADIUSClientAssignments)...)

	return validationErrors
}

func validateDatabase(tenantIdentifier string, database model.Database) []error {
	var validationErrors []error

	if database.Engine == "" {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("tenant %q: database engine must be specified", tenantIdentifier),
		)
	} else if database.Engine != "mysql" {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("tenant %q: database engine %q is not supported", tenantIdentifier, database.Engine),
		)
	}

	deployment := database.Deployment
	if deployment == "" {
		deployment = "container"
	}

	switch deployment {
	case "container":
		// Host and port are determined by the deployment mechanism.

	case "proxysql", "external":
		if database.Host == "" {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"tenant %q: database host must be specified for %s deployment",
					tenantIdentifier,
					deployment,
				),
			)
		}

		if database.Port < 1 || database.Port > 65535 {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"tenant %q: database port must be between 1 and 65535 for %s deployment",
					tenantIdentifier,
					deployment,
				),
			)
		}

	default:
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"tenant %q: database deployment %q is not supported",
				tenantIdentifier,
				deployment,
			),
		)
	}

	if database.Database == "" {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("tenant %q: database name must be specified", tenantIdentifier),
		)
	}

	if database.Username == "" {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("tenant %q: database username must be specified", tenantIdentifier),
		)
	}

	if database.Password == "" {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("tenant %q: database password must be specified", tenantIdentifier),
		)
	}

	return validationErrors
}

func validateRADIUSServer(tenantIdentifier string, server model.RADIUSServer, templateLoader templates.Loader) []error {
	var validationErrors []error
	if server.Version == "" {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: radius server version must be specified", tenantIdentifier))
	} else if !templateLoader.SupportsVersion(server.Version) {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: radius server version %q is not supported", tenantIdentifier, server.Version))
	}
	if server.AuthenticationPort < 1 || server.AuthenticationPort > 65535 {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: radius server authentication_port must be between 1 and 65535", tenantIdentifier))
	}
	if server.AccountingPort < 1 || server.AccountingPort > 65535 {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: radius server accounting_port must be between 1 and 65535", tenantIdentifier))
	}
	if server.COAPort < 1 || server.COAPort > 65535 {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: radius server coa_port must be between 1 and 65535", tenantIdentifier))
	}

	return validationErrors
}
func validateNASAssignments(tenantIdentifier string, assignments map[string]model.NASAssignment) []error {
	var validationErrors []error
	for _, identifier := range sortedKeys(assignments) {
		validationErrors = append(validationErrors, validateNASAssignment(tenantIdentifier, identifier, assignments[identifier])...)
	}

	return validationErrors
}

func validateNASAssignment(tenantIdentifier, identifier string, assignment model.NASAssignment) []error {
	var validationErrors []error
	if assignment.CredentialProfile == "" {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: nas assignment %q: credential_profile must be specified", tenantIdentifier, identifier))
	}
	if assignment.AccountingProfile == "" {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: nas assignment %q: accounting_profile must be specified", tenantIdentifier, identifier))
	}
	if assignment.MonitoringProfile == "" {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: nas assignment %q: monitoring_profile must be specified", tenantIdentifier, identifier))
	}
	validationErrors = append(validationErrors, validateRequireMessageAuthenticator(
		fmt.Sprintf("tenant %q: nas assignment %q", tenantIdentifier, identifier),
		assignment.RequireMessageAuthenticator,
	)...)

	return validationErrors
}

func validateTrustedRADIUSClientAssignments(tenantIdentifier string, assignments map[string]model.TrustedRADIUSClientAssignment) []error {
	var validationErrors []error
	for _, identifier := range sortedKeys(assignments) {
		validationErrors = append(validationErrors, validateTrustedRADIUSClientAssignment(tenantIdentifier, identifier, assignments[identifier])...)
	}

	return validationErrors
}

func validateTrustedRADIUSClientAssignment(tenantIdentifier, identifier string, assignment model.TrustedRADIUSClientAssignment) []error {
	var validationErrors []error
	if assignment.CredentialProfile == "" {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: trusted radius client assignment %q: credential_profile must be specified", tenantIdentifier, identifier))
	}
	validationErrors = append(validationErrors, validateRequireMessageAuthenticator(
		fmt.Sprintf("tenant %q: trusted radius client assignment %q", tenantIdentifier, identifier),
		assignment.RequireMessageAuthenticator,
	)...)

	return validationErrors
}

func validateRequireMessageAuthenticator(prefix string, value *model.RequireMessageAuthenticator) []error {
	if value == nil {
		return nil
	}

	switch *value {
	case model.RequireMessageAuthenticatorAuto,
		model.RequireMessageAuthenticatorYes,
		model.RequireMessageAuthenticatorNo:
		return nil
	default:
		return []error{fmt.Errorf(
			"%s: require_message_authenticator must be one of auto, yes, or no",
			prefix,
		)}
	}
}
