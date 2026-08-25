package generator

import (
	"sort"
	"time"

	"github.com/gobcn/radius-director/internal/model"
)

// Generate builds an intermediate FreeRADIUS configuration from a validated model.
func Generate(configuration model.Configuration) Configuration {
	generated := Configuration{
		Tenants: make([]Tenant, 0, len(configuration.Tenants)),
	}

	for _, tenantIdentifier := range sortedKeys(configuration.Tenants) {
		tenant := configuration.Tenants[tenantIdentifier]
		authenticationProfile := configuration.GlobalObjects.AuthenticationProfiles[tenant.AuthenticationProfile]
		deploymentProfile := configuration.GlobalObjects.DeploymentProfiles[tenant.DeploymentProfile]
		databaseHost, databasePort := databaseEndpoint(
			tenantIdentifier,
			tenant.Database,
		)

		var proxySQL *ProxySQL

		if databaseDeployment(tenant.Database) == "proxysql" {
			proxySQL = &ProxySQL{
				BackendHost: tenant.Database.Host,
				BackendPort: tenant.Database.Port,
			}
		}
		generatedTenant := Tenant{
			Identifier: tenantIdentifier,
			AuthenticationPolicy: AuthenticationPolicy{
				SimultaneousUse: authenticationProfile.SimultaneousUse,
			},
			FreeRADIUSClients:  make([]FreeRADIUSClient, 0, len(tenant.NASAssignments)+len(tenant.TrustedRADIUSClientAssignments)),
			HomeServers:        make([]HomeServer, 0, len(tenant.NASAssignments)),
			AccountingPolicies: make([]NASAccountingPolicy, 0, len(tenant.NASAssignments)),
			DatabaseDeployment: databaseDeployment(tenant.Database),
			SQL: SQL{
				Engine:   tenant.Database.Engine,
				Host:     databaseHost,
				Port:     databasePort,
				Database: tenant.Database.Database,
				Username: tenant.Database.Username,
				Password: tenant.Database.Password,
			},
			ProxySQL: proxySQL,
			RADIUSServer: RADIUSServer{
				Version:            tenant.RADIUSServer.Version,
				AuthenticationPort: tenant.RADIUSServer.AuthenticationPort,
				AccountingPort:     tenant.RADIUSServer.AccountingPort,
				COAPort:            tenant.RADIUSServer.COAPort,
			},
			Template: deploymentProfile.Template,
			Overlays: deploymentProfile.Overlays,
			Remove:   deploymentProfile.Remove,
		}

		for _, nasDeviceIdentifier := range sortedKeys(tenant.NASAssignments) {
			assignment := tenant.NASAssignments[nasDeviceIdentifier]
			nasDevice := configuration.GlobalObjects.NASDevices[nasDeviceIdentifier]
			credentialProfile := configuration.GlobalObjects.CredentialProfiles[assignment.CredentialProfile]
			accountingProfile := configuration.GlobalObjects.AccountingProfiles[assignment.AccountingProfile]

			var staleSessionTimeout *time.Duration
			if accountingProfile.StaleSessionTimeout != "" {
				parsedTimeout, _ := time.ParseDuration(accountingProfile.StaleSessionTimeout)
				staleSessionTimeout = &parsedTimeout
			}

			requireMessageAuthenticator := messageAuthenticatorValue(assignment.RequireMessageAuthenticator)

			generatedTenant.FreeRADIUSClients = append(generatedTenant.FreeRADIUSClients, FreeRADIUSClient{
				Identifier:                  nasDeviceIdentifier,
				IPAddress:                   nasDevice.IPAddress,
				SharedSecret:                credentialProfile.SharedSecret,
				Vendor:                      nasDevice.Vendor,
				RequireMessageAuthenticator: requireMessageAuthenticator,
			})
			generatedTenant.HomeServers = append(generatedTenant.HomeServers, HomeServer{
				Identifier:   nasDeviceIdentifier,
				IPAddress:    nasDevice.IPAddress,
				SharedSecret: credentialProfile.SharedSecret,
			})
			generatedTenant.AccountingPolicies = append(generatedTenant.AccountingPolicies, NASAccountingPolicy{
				NASDeviceIdentifier: nasDeviceIdentifier,
				IPAddress:           nasDevice.IPAddress,
				StaleSessionTimeout: staleSessionTimeout,
			})
		}

		for _, trustedRADIUSClientIdentifier := range sortedKeys(tenant.TrustedRADIUSClientAssignments) {
			assignment := tenant.TrustedRADIUSClientAssignments[trustedRADIUSClientIdentifier]
			trustedRADIUSClient := configuration.GlobalObjects.TrustedRADIUSClients[trustedRADIUSClientIdentifier]
			credentialProfile := configuration.GlobalObjects.CredentialProfiles[assignment.CredentialProfile]
			requireMessageAuthenticator := messageAuthenticatorValue(assignment.RequireMessageAuthenticator)
			generatedTenant.FreeRADIUSClients = append(generatedTenant.FreeRADIUSClients, FreeRADIUSClient{
				Identifier:                  trustedRADIUSClientIdentifier,
				IPAddress:                   trustedRADIUSClient.IPAddress,
				SharedSecret:                credentialProfile.SharedSecret,
				RequireMessageAuthenticator: requireMessageAuthenticator,
			})
		}
		sort.SliceStable(generatedTenant.FreeRADIUSClients, func(left, right int) bool {
			return generatedTenant.FreeRADIUSClients[left].Identifier < generatedTenant.FreeRADIUSClients[right].Identifier
		})

		generated.Tenants = append(generated.Tenants, generatedTenant)
	}

	return generated
}

func messageAuthenticatorValue(value *model.RequireMessageAuthenticator) *string {
	if value == nil {
		return nil
	}

	configured := string(*value)
	return &configured
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

func databaseDeployment(database model.Database) string {
	if database.Deployment == "" {
		return "container"
	}

	return database.Deployment
}

func databaseEndpoint(tenantIdentifier string, database model.Database) (string, int) {
	deployment := databaseDeployment(database)

	switch deployment {
	case "container":
		return "database-" + tenantIdentifier, 3306

	case "proxysql":
		return "proxysql", 6033

	case "external":
		return database.Host, database.Port

	default:
		return database.Host, database.Port
	}
}
