package validation

import (
	"fmt"

	"github.com/gobcn/radius-director/internal/model"
)

func validateReferences(configuration model.Configuration) []error {
	var validationErrors []error
	for _, identifier := range sortedKeys(configuration.Tenants) {
		validationErrors = append(validationErrors, validateTenantReferences(identifier, configuration.Tenants[identifier], configuration.GlobalObjects)...)
	}

	return validationErrors
}

func validateTenantReferences(tenantIdentifier string, tenant model.Tenant, globalObjects model.GlobalObjects) []error {
	var validationErrors []error
	if tenant.AuthenticationProfile != "" {
		if _, exists := globalObjects.AuthenticationProfiles[tenant.AuthenticationProfile]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("tenant %q: authentication profile %q does not exist", tenantIdentifier, tenant.AuthenticationProfile))
		}
	}
	if tenant.DeploymentProfile != "" {
		if _, exists := globalObjects.DeploymentProfiles[tenant.DeploymentProfile]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("tenant %q: deployment profile %q does not exist", tenantIdentifier, tenant.DeploymentProfile))
		}
	}
	for _, identifier := range sortedKeys(tenant.NASAssignments) {
		validationErrors = append(validationErrors, validateNASAssignmentReferences(tenantIdentifier, identifier, tenant.NASAssignments[identifier], globalObjects)...)
	}
	for _, identifier := range sortedKeys(tenant.TrustedRADIUSClientAssignments) {
		validationErrors = append(validationErrors, validateTrustedRADIUSClientAssignmentReferences(tenantIdentifier, identifier, tenant.TrustedRADIUSClientAssignments[identifier], globalObjects)...)
	}

	return validationErrors
}

func validateNASAssignmentReferences(tenantIdentifier, identifier string, assignment model.NASAssignment, globalObjects model.GlobalObjects) []error {
	var validationErrors []error
	if _, exists := globalObjects.NASDevices[identifier]; !exists {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: nas assignment %q: nas device %q does not exist", tenantIdentifier, identifier, identifier))
	}
	if assignment.CredentialProfile != "" {
		if _, exists := globalObjects.CredentialProfiles[assignment.CredentialProfile]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("tenant %q: nas assignment %q: credential profile %q does not exist", tenantIdentifier, identifier, assignment.CredentialProfile))
		}
	}
	if assignment.AccountingProfile != "" {
		if _, exists := globalObjects.AccountingProfiles[assignment.AccountingProfile]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("tenant %q: nas assignment %q: accounting profile %q does not exist", tenantIdentifier, identifier, assignment.AccountingProfile))
		}
	}
	if assignment.MonitoringProfile != "" {
		if _, exists := globalObjects.MonitoringProfiles[assignment.MonitoringProfile]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("tenant %q: nas assignment %q: monitoring profile %q does not exist", tenantIdentifier, identifier, assignment.MonitoringProfile))
		}
	}

	return validationErrors
}

func validateTrustedRADIUSClientAssignmentReferences(tenantIdentifier, identifier string, assignment model.TrustedRADIUSClientAssignment, globalObjects model.GlobalObjects) []error {
	var validationErrors []error
	if _, exists := globalObjects.TrustedRADIUSClients[identifier]; !exists {
		validationErrors = append(validationErrors, fmt.Errorf("tenant %q: trusted radius client assignment %q: trusted radius client %q does not exist", tenantIdentifier, identifier, identifier))
	}
	if assignment.CredentialProfile != "" {
		if _, exists := globalObjects.CredentialProfiles[assignment.CredentialProfile]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("tenant %q: trusted radius client assignment %q: credential profile %q does not exist", tenantIdentifier, identifier, assignment.CredentialProfile))
		}
	}

	return validationErrors
}
