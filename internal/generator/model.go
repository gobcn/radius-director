// Package generator builds intermediate FreeRADIUS configuration models.
package generator

import "time"

// Configuration is an intermediate representation of generated FreeRADIUS
// configuration, grouped by tenant.
type Configuration struct {
	Tenants []Tenant
}

// Tenant contains generated configuration for one RADIUS Director tenant.
type Tenant struct {
	Identifier           string
	FreeRADIUSClients    []FreeRADIUSClient
	HomeServers          []HomeServer
	AuthenticationPolicy AuthenticationPolicy
	AccountingPolicies   []NASAccountingPolicy
	DatabaseDeployment   string
	SQL                  SQL
	ProxySQL             *ProxySQL
	RADIUSServer         RADIUSServer
	Template             string
	Overlays             []string
	Remove               []string
}

// AuthenticationPolicy contains resolved tenant-wide subscriber authentication policy.
type AuthenticationPolicy struct {
	SimultaneousUse *int
}

// FreeRADIUSClient contains the information needed to render a FreeRADIUS
// client definition.
type FreeRADIUSClient struct {
	Identifier                  string
	IPAddress                   string
	SharedSecret                string
	Vendor                      string
	RequireMessageAuthenticator *string
}

// HomeServer contains the information needed to render a FreeRADIUS home
// server definition.
type HomeServer struct {
	Identifier   string
	IPAddress    string
	SharedSecret string
}

// NASAccountingPolicy contains resolved accounting maintenance policy for one
// NAS assignment. A nil StaleSessionTimeout means stale-session cleanup is
// disabled for that assignment.
type NASAccountingPolicy struct {
	NASDeviceIdentifier string
	IPAddress           string
	StaleSessionTimeout *time.Duration
}

// SQL contains the information needed to render a FreeRADIUS sql module.
type SQL struct {
	Engine   string
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

// ProxySQL contains the information needed to configure a ProxySQL
// instance for a tenant.
type ProxySQL struct {
	BackendHost string
	BackendPort int
}

// RADIUSServer contains the information needed to render a FreeRADIUS server.
type RADIUSServer struct {
	Version            string
	AuthenticationPort int
	AccountingPort     int
	COAPort            int
}
