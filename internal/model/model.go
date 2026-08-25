// Package model defines the RADIUS Director configuration object model.
package model

// Configuration represents a root RADIUS Director configuration document.
type Configuration struct {
	GlobalObjects GlobalObjects     `yaml:"global_objects"`
	Tenants       map[string]Tenant `yaml:"tenants"`
}

// GlobalObjects contains the globally reusable configuration objects.
type GlobalObjects struct {
	CredentialProfiles     map[string]CredentialProfile     `yaml:"credential_profiles"`
	AuthenticationProfiles map[string]AuthenticationProfile `yaml:"authentication_profiles"`
	AccountingProfiles     map[string]AccountingProfile     `yaml:"accounting_profiles"`
	MonitoringProfiles     map[string]MonitoringProfile     `yaml:"monitoring_profiles"`
	DeploymentProfiles     map[string]DeploymentProfile     `yaml:"deployment_profiles"`
	NASDevices             map[string]NASDevice             `yaml:"nas_devices"`
	TrustedRADIUSClients   map[string]TrustedRADIUSClient   `yaml:"trusted_radius_clients"`
}

// CredentialProfile defines the shared RADIUS credentials used to communicate
// with a NAS device.
type CredentialProfile struct {
	SharedSecret string `yaml:"shared_secret"`
}

// AuthenticationProfile defines tenant-wide subscriber authentication policy.
type AuthenticationProfile struct {
	SimultaneousUse *int `yaml:"simultaneous_use"`
}

// AccountingProfile defines accounting behaviour.
type AccountingProfile struct {
	StaleSessionTimeout string `yaml:"stale_session_timeout"`
}

// MonitoringProfile defines operational monitoring.
//
// Its properties are implementation-specific and are not defined yet.
type MonitoringProfile struct{}

// DeploymentProfile defines a specific deployment configuration.
type DeploymentProfile struct {
	Template string   `yaml:"template"`
	Overlays []string `yaml:"overlays"`
	Remove   []string `yaml:"remove"`
}

// NASDevice represents a physical or virtual RADIUS client.
type NASDevice struct {
	IPAddress string `yaml:"ip_address"`
	Vendor    string `yaml:"vendor"`
}

// TrustedRADIUSClient represents a trusted RADIUS client that is not a NAS
// device.
type TrustedRADIUSClient struct {
	IPAddress string `yaml:"ip_address"`
}

// Tenant represents an independent RADIUS deployment.
type Tenant struct {
	AuthenticationProfile          string                                   `yaml:"authentication_profile"`
	DeploymentProfile              string                                   `yaml:"deployment_profile"`
	Database                       Database                                 `yaml:"database"`
	RADIUSServer                   RADIUSServer                             `yaml:"radius_server"`
	NASAssignments                 map[string]NASAssignment                 `yaml:"nas_assignments"`
	TrustedRADIUSClientAssignments map[string]TrustedRADIUSClientAssignment `yaml:"trusted_radius_client_assignments"`
}

// Database defines the primary database used by a tenant.
type Database struct {
	Engine     string `yaml:"engine"`
	Deployment string `yaml:"deployment"`
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Database   string `yaml:"database"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
}

// RADIUSServer represents a FreeRADIUS instance.
type RADIUSServer struct {
	Version            string `yaml:"version"`
	AuthenticationPort int    `yaml:"authentication_port"`
	AccountingPort     int    `yaml:"accounting_port"`
	COAPort            int    `yaml:"coa_port"`
}

// NASAssignment defines how a tenant uses a NAS device.
type NASAssignment struct {
	CredentialProfile           string                       `yaml:"credential_profile"`
	AccountingProfile           string                       `yaml:"accounting_profile"`
	MonitoringProfile           string                       `yaml:"monitoring_profile"`
	RequireMessageAuthenticator *RequireMessageAuthenticator `yaml:"require_message_authenticator"`
}

// TrustedRADIUSClientAssignment defines how a tenant uses a trusted RADIUS
// client.
type TrustedRADIUSClientAssignment struct {
	CredentialProfile           string                       `yaml:"credential_profile"`
	RequireMessageAuthenticator *RequireMessageAuthenticator `yaml:"require_message_authenticator"`
}

// RequireMessageAuthenticator controls FreeRADIUS Message-Authenticator
// enforcement for an inbound RADIUS client.
type RequireMessageAuthenticator string

const (
	RequireMessageAuthenticatorAuto RequireMessageAuthenticator = "auto"
	RequireMessageAuthenticatorYes  RequireMessageAuthenticator = "yes"
	RequireMessageAuthenticatorNo   RequireMessageAuthenticator = "no"
)
