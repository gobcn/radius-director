package validation

import (
	"fmt"

	"github.com/gobcn/radius-director/internal/model"
)

func validateRelationships(configuration model.Configuration) []error {
	var validationErrors []error

	validationErrors = append(
		validationErrors,
		validateProxySQLUsernames(configuration)...,
	)

	for _, identifier := range sortedKeys(configuration.Tenants) {
		validationErrors = append(
			validationErrors,
			validateTenantRelationships(
				identifier,
				configuration.Tenants[identifier],
				configuration.GlobalObjects,
			)...,
		)
	}

	return validationErrors
}

func validateTenantRelationships(tenantIdentifier string, tenant model.Tenant, globalObjects model.GlobalObjects) []error {
	var validationErrors []error
	for _, identifier := range sortedKeys(tenant.NASAssignments) {
		if _, exists := globalObjects.NASDevices[identifier]; !exists {
			continue
		}
		if _, exists := globalObjects.TrustedRADIUSClients[identifier]; !exists {
			continue
		}
		if _, exists := tenant.TrustedRADIUSClientAssignments[identifier]; !exists {
			continue
		}

		validationErrors = append(validationErrors, fmt.Errorf(
			"tenant %q: nas device %q and trusted radius client %q both generate FreeRADIUS client %q",
			tenantIdentifier,
			identifier,
			identifier,
			identifier,
		))
	}

	return validationErrors
}

func validateProxySQLUsernames(configuration model.Configuration) []error {
	var validationErrors []error
	usernames := make(map[string]string)

	for _, identifier := range sortedKeys(configuration.Tenants) {
		tenant := configuration.Tenants[identifier]

		deployment := tenant.Database.Deployment
		if deployment == "" {
			deployment = "container"
		}

		if deployment != "proxysql" {
			continue
		}

		username := tenant.Database.Username
		if username == "" {
			continue
		}

		if firstTenant, exists := usernames[username]; exists {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"tenants %q and %q both use ProxySQL database username %q",
					firstTenant,
					identifier,
					username,
				),
			)
			continue
		}

		usernames[username] = identifier
	}

	return validationErrors
}
