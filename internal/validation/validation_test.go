package validation

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gobcn/radius-director/internal/model"
	"github.com/gobcn/radius-director/internal/templates"
)

func testTemplateLoader(t *testing.T) templates.Loader {
	t.Helper()

	directory, err := filepath.Abs(filepath.Join("..", "..", "templates"))
	if err != nil {
		t.Fatalf("resolve template directory: %v", err)
	}

	return templates.NewLoader(os.DirFS(directory))
}

func TestValidate(t *testing.T) {
	one := 1
	configuration := model.Configuration{
		GlobalObjects: model.GlobalObjects{
			CredentialProfiles: map[string]model.CredentialProfile{
				"default": {SharedSecret: "secret"},
			},
			AuthenticationProfiles: map[string]model.AuthenticationProfile{
				"default": {SimultaneousUse: &one},
			},
			AccountingProfiles: map[string]model.AccountingProfile{
				"default": {},
			},
			MonitoringProfiles: map[string]model.MonitoringProfile{
				"default": {},
			},
			DeploymentProfiles: map[string]model.DeploymentProfile{
				"default": {Template: "default"},
			},
			NASDevices: map[string]model.NASDevice{
				"core": {IPAddress: "10.10.10.1", Vendor: "mikrotik"},
			},
			TrustedRADIUSClients: map[string]model.TrustedRADIUSClient{
				"monitoring": {IPAddress: "10.10.10.2"},
			},
		},
		Tenants: map[string]model.Tenant{
			"customer-a": {
				AuthenticationProfile: "default",
				DeploymentProfile:     "default",
				Database: model.Database{
					Engine:   "mysql",
					Host:     "db.example.com",
					Port:     3306,
					Database: "radius",
					Username: "radius",
					Password: "secret",
				},
				RADIUSServer: model.RADIUSServer{
					Version:            "3.2.10",
					AuthenticationPort: 1812,
					AccountingPort:     1813,
					COAPort:            3799,
				},
				NASAssignments: map[string]model.NASAssignment{
					"core": {
						CredentialProfile: "default",
						AccountingProfile: "default",
						MonitoringProfile: "default",
					},
				},
			},
		},
	}

	if err := Validate(configuration, testTemplateLoader(t)); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestValidateNASAssignment(t *testing.T) {
	validAssignment := model.NASAssignment{
		CredentialProfile: "default",
		AccountingProfile: "default",
		MonitoringProfile: "default",
	}

	tests := []struct {
		name       string
		assignment model.NASAssignment
		wantErrs   []string
	}{
		{
			name:       "valid NAS Assignment",
			assignment: validAssignment,
		},
		{
			name: "credential profile missing",
			assignment: model.NASAssignment{
				AccountingProfile: validAssignment.AccountingProfile,
				MonitoringProfile: validAssignment.MonitoringProfile,
			},
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": credential_profile must be specified`,
			},
		},
		{
			name: "accounting profile missing",
			assignment: model.NASAssignment{
				CredentialProfile: validAssignment.CredentialProfile,
				MonitoringProfile: validAssignment.MonitoringProfile,
			},
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": accounting_profile must be specified`,
			},
		},
		{
			name: "monitoring profile missing",
			assignment: model.NASAssignment{
				CredentialProfile: validAssignment.CredentialProfile,
				AccountingProfile: validAssignment.AccountingProfile,
			},
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": monitoring_profile must be specified`,
			},
		},
		{
			name: "multiple properties missing",
			assignment: model.NASAssignment{
				MonitoringProfile: validAssignment.MonitoringProfile,
			},
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": credential_profile must be specified`,
				`tenant "customer-a": nas assignment "core": accounting_profile must be specified`,
			},
		},
		{
			name: "all properties missing",
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": credential_profile must be specified`,
				`tenant "customer-a": nas assignment "core": accounting_profile must be specified`,
				`tenant "customer-a": nas assignment "core": monitoring_profile must be specified`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateNASAssignment("customer-a", "core", test.assignment)
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateNASAssignment() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateNASAssignment() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateNASAssignmentReferences(t *testing.T) {
	one := 1
	globalObjects := model.GlobalObjects{
		CredentialProfiles: map[string]model.CredentialProfile{
			"default": {},
		},
		AuthenticationProfiles: map[string]model.AuthenticationProfile{
			"default": {SimultaneousUse: &one},
		},
		AccountingProfiles: map[string]model.AccountingProfile{
			"default": {},
		},
		MonitoringProfiles: map[string]model.MonitoringProfile{
			"default": {},
		},
		NASDevices: map[string]model.NASDevice{
			"core": {},
		},
	}
	validAssignment := model.NASAssignment{
		CredentialProfile: "default",
		AccountingProfile: "default",
		MonitoringProfile: "default",
	}

	tests := []struct {
		name       string
		assignment model.NASAssignment
		wantErrs   []string
	}{
		{
			name:       "valid references",
			assignment: validAssignment,
		},
		{
			name: "credential profile missing",
			assignment: model.NASAssignment{
				CredentialProfile: "missing-credential-profile",
				AccountingProfile: validAssignment.AccountingProfile,
				MonitoringProfile: validAssignment.MonitoringProfile,
			},
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": credential profile "missing-credential-profile" does not exist`,
			},
		},
		{
			name: "accounting profile missing",
			assignment: model.NASAssignment{
				CredentialProfile: validAssignment.CredentialProfile,
				AccountingProfile: "missing-accounting-profile",
				MonitoringProfile: validAssignment.MonitoringProfile,
			},
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": accounting profile "missing-accounting-profile" does not exist`,
			},
		},
		{
			name: "monitoring profile missing",
			assignment: model.NASAssignment{
				CredentialProfile: validAssignment.CredentialProfile,
				AccountingProfile: validAssignment.AccountingProfile,
				MonitoringProfile: "missing-monitoring-profile",
			},
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": monitoring profile "missing-monitoring-profile" does not exist`,
			},
		},
		{
			name: "multiple references missing",
			assignment: model.NASAssignment{
				CredentialProfile: validAssignment.CredentialProfile,
				AccountingProfile: "missing-accounting-profile",
				MonitoringProfile: validAssignment.MonitoringProfile,
			},
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": accounting profile "missing-accounting-profile" does not exist`,
			},
		},
		{
			name: "all references missing",
			assignment: model.NASAssignment{
				CredentialProfile: "missing-credential-profile",
				AccountingProfile: "missing-accounting-profile",
				MonitoringProfile: "missing-monitoring-profile",
			},
			wantErrs: []string{
				`tenant "customer-a": nas assignment "core": credential profile "missing-credential-profile" does not exist`,
				`tenant "customer-a": nas assignment "core": accounting profile "missing-accounting-profile" does not exist`,
				`tenant "customer-a": nas assignment "core": monitoring profile "missing-monitoring-profile" does not exist`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateNASAssignmentReferences("customer-a", "core", test.assignment, globalObjects)
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateNASAssignmentReferences() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateNASAssignmentReferences() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateAssignmentObjectReferencesUseAssignmentKeys(t *testing.T) {
	globals := model.GlobalObjects{
		NASDevices:             map[string]model.NASDevice{},
		TrustedRADIUSClients:   map[string]model.TrustedRADIUSClient{},
		CredentialProfiles:     map[string]model.CredentialProfile{"default": {}},
		AuthenticationProfiles: map[string]model.AuthenticationProfile{},
		AccountingProfiles:     map[string]model.AccountingProfile{},
		MonitoringProfiles:     map[string]model.MonitoringProfile{},
	}

	nasErrors := validateNASAssignmentReferences("customer-a", "missing-nas", model.NASAssignment{}, globals)
	if len(nasErrors) == 0 || nasErrors[0].Error() != `tenant "customer-a": nas assignment "missing-nas": nas device "missing-nas" does not exist` {
		t.Fatalf("NAS assignment errors = %v", nasErrors)
	}

	trustedErrors := validateTrustedRADIUSClientAssignmentReferences("customer-a", "missing-client", model.TrustedRADIUSClientAssignment{}, globals)
	if len(trustedErrors) == 0 || trustedErrors[0].Error() != `tenant "customer-a": trusted radius client assignment "missing-client": trusted radius client "missing-client" does not exist` {
		t.Fatalf("trusted RADIUS client assignment errors = %v", trustedErrors)
	}
}

func TestValidateRequireMessageAuthenticator(t *testing.T) {
	validValues := []model.RequireMessageAuthenticator{
		model.RequireMessageAuthenticatorAuto,
		model.RequireMessageAuthenticatorYes,
		model.RequireMessageAuthenticatorNo,
	}

	for _, value := range validValues {
		value := value
		t.Run(string(value), func(t *testing.T) {
			nasAssignment := model.NASAssignment{RequireMessageAuthenticator: &value}
			if errs := validateNASAssignment("customer-a", "core", nasAssignment); len(errs) != 3 {
				t.Fatalf("validateNASAssignment() errors = %v, want only the three required-profile errors", errs)
			}

			trustedAssignment := model.TrustedRADIUSClientAssignment{CredentialProfile: "default", RequireMessageAuthenticator: &value}
			if errs := validateTrustedRADIUSClientAssignment("customer-a", "sonar", trustedAssignment); len(errs) != 0 {
				t.Fatalf("validateTrustedRADIUSClientAssignment() errors = %v, want none", errs)
			}
		})
	}

	invalid := model.RequireMessageAuthenticator("sometimes")
	for _, test := range []struct {
		name string
		errs []error
	}{
		{"NAS assignment", validateNASAssignment("customer-a", "core", model.NASAssignment{RequireMessageAuthenticator: &invalid})},
		{"trusted RADIUS client assignment", validateTrustedRADIUSClientAssignment("customer-a", "sonar", model.TrustedRADIUSClientAssignment{CredentialProfile: "default", RequireMessageAuthenticator: &invalid})},
	} {
		t.Run(test.name, func(t *testing.T) {
			found := false
			for _, err := range test.errs {
				if strings.Contains(err.Error(), "require_message_authenticator must be one of auto, yes, or no") {
					found = true
				}
			}
			if !found {
				t.Fatalf("errors = %v, want require_message_authenticator validation error", test.errs)
			}
		})
	}
}

func TestValidateTenantRelationships(t *testing.T) {
	globalObjects := model.GlobalObjects{
		NASDevices: map[string]model.NASDevice{
			"core":  {},
			"sonar": {},
		},
		TrustedRADIUSClients: map[string]model.TrustedRADIUSClient{
			"sonar": {},
		},
	}

	tests := []struct {
		name     string
		tenant   model.Tenant
		wantErrs []string
	}{
		{
			name: "distinct FreeRADIUS client names",
			tenant: model.Tenant{
				NASAssignments: map[string]model.NASAssignment{
					"core": {},
				},
				TrustedRADIUSClientAssignments: map[string]model.TrustedRADIUSClientAssignment{
					"sonar": {},
				},
			},
		},
		{
			name: "cross-type generated client name collision",
			tenant: model.Tenant{
				NASAssignments: map[string]model.NASAssignment{
					"sonar": {},
				},
				TrustedRADIUSClientAssignments: map[string]model.TrustedRADIUSClientAssignment{
					"sonar": {},
				},
			},
			wantErrs: []string{`tenant "customer-a": nas device "sonar" and trusted radius client "sonar" both generate FreeRADIUS client "sonar"`},
		},
		{
			name: "missing referenced objects do not produce collision errors",
			tenant: model.Tenant{
				NASAssignments: map[string]model.NASAssignment{
					"missing": {},
				},
				TrustedRADIUSClientAssignments: map[string]model.TrustedRADIUSClientAssignment{
					"missing": {},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateTenantRelationships("customer-a", test.tenant, globalObjects)
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateTenantRelationships() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateTenantRelationships() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateProxySQLUsernames(t *testing.T) {
	tests := []struct {
		name          string
		configuration model.Configuration
		wantErrs      []string
	}{
		{
			name: "unique ProxySQL usernames",
			configuration: model.Configuration{
				Tenants: map[string]model.Tenant{
					"customer-a": {
						Database: model.Database{
							Deployment: "proxysql",
							Username:   "radius_a",
						},
					},
					"customer-b": {
						Database: model.Database{
							Deployment: "proxysql",
							Username:   "radius_b",
						},
					},
				},
			},
		},
		{
			name: "duplicate ProxySQL username",
			configuration: model.Configuration{
				Tenants: map[string]model.Tenant{
					"customer-a": {
						Database: model.Database{
							Deployment: "proxysql",
							Username:   "radius",
						},
					},
					"customer-b": {
						Database: model.Database{
							Deployment: "proxysql",
							Username:   "radius",
						},
					},
				},
			},
			wantErrs: []string{
				`tenants "customer-a" and "customer-b" both use ProxySQL database username "radius"`,
			},
		},
		{
			name: "duplicate username allowed for external databases",
			configuration: model.Configuration{
				Tenants: map[string]model.Tenant{
					"customer-a": {
						Database: model.Database{
							Deployment: "external",
							Username:   "radius",
						},
					},
					"customer-b": {
						Database: model.Database{
							Deployment: "external",
							Username:   "radius",
						},
					},
				},
			},
		},
		{
			name: "duplicate username allowed for container databases",
			configuration: model.Configuration{
				Tenants: map[string]model.Tenant{
					"customer-a": {
						Database: model.Database{
							Deployment: "container",
							Username:   "radius",
						},
					},
					"customer-b": {
						Database: model.Database{
							Deployment: "container",
							Username:   "radius",
						},
					},
				},
			},
		},
		{
			name: "duplicate ProxySQL username reports all conflicts",
			configuration: model.Configuration{
				Tenants: map[string]model.Tenant{
					"customer-a": {
						Database: model.Database{
							Deployment: "proxysql",
							Username:   "radius",
						},
					},
					"customer-b": {
						Database: model.Database{
							Deployment: "proxysql",
							Username:   "radius",
						},
					},
					"customer-c": {
						Database: model.Database{
							Deployment: "proxysql",
							Username:   "radius",
						},
					},
				},
			},
			wantErrs: []string{
				`tenants "customer-a" and "customer-b" both use ProxySQL database username "radius"`,
				`tenants "customer-a" and "customer-c" both use ProxySQL database username "radius"`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateProxySQLUsernames(test.configuration)

			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf(
					"validateProxySQLUsernames() returned %d errors, want %d",
					len(validationErrors),
					len(test.wantErrs),
				)
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf(
						"validateProxySQLUsernames() error = %q, want %q",
						got,
						wantErr,
					)
				}
			}
		})
	}
}

func TestValidateTenant(t *testing.T) {
	validTenant := model.Tenant{
		AuthenticationProfile: "default",
		DeploymentProfile:     "default",
		Database: model.Database{
			Engine:   "mysql",
			Host:     "db.example.com",
			Port:     3306,
			Database: "radius",
			Username: "radius",
			Password: "secret",
		},
		RADIUSServer: model.RADIUSServer{
			Version:            "3.2.10",
			AuthenticationPort: 1812,
			AccountingPort:     1813,
			COAPort:            3799,
		},
		NASAssignments: map[string]model.NASAssignment{
			"core": {
				CredentialProfile: "default",
				AccountingProfile: "default",
				MonitoringProfile: "default",
			},
		},
	}

	tests := []struct {
		name     string
		tenant   model.Tenant
		wantErrs []string
	}{
		{
			name:   "valid tenant",
			tenant: validTenant,
		},
		{
			name: "database missing",
			tenant: model.Tenant{
				AuthenticationProfile: validTenant.AuthenticationProfile,
				DeploymentProfile:     validTenant.DeploymentProfile,
				RADIUSServer:          validTenant.RADIUSServer,
				NASAssignments:        validTenant.NASAssignments,
			},
			wantErrs: []string{
				`tenant "customer-a": exactly one database must be defined`,
			},
		},
		{
			name: "RADIUS Server missing",
			tenant: model.Tenant{
				AuthenticationProfile: validTenant.AuthenticationProfile,
				DeploymentProfile:     validTenant.DeploymentProfile,
				Database:              validTenant.Database,
				NASAssignments:        validTenant.NASAssignments,
			},
			wantErrs: []string{
				`tenant "customer-a": exactly one radius server must be defined`,
			},
		},
		{
			name: "NAS assignments missing",
			tenant: model.Tenant{
				AuthenticationProfile: validTenant.AuthenticationProfile,
				DeploymentProfile:     validTenant.DeploymentProfile,
				Database:              validTenant.Database,
				RADIUSServer:          validTenant.RADIUSServer,
			},
			wantErrs: []string{
				`tenant "customer-a": at least one nas assignment must be defined`,
			},
		},
		{
			name: "deployment profile missing",
			tenant: model.Tenant{
				AuthenticationProfile: validTenant.AuthenticationProfile,
				Database:              validTenant.Database,
				RADIUSServer:          validTenant.RADIUSServer,
				NASAssignments:        validTenant.NASAssignments,
			},
			wantErrs: []string{
				`tenant "customer-a": deployment_profile must be specified`,
			},
		},
		{
			name: "all required tenant objects missing",
			wantErrs: []string{
				`tenant "customer-a": authentication_profile must be specified`,
				`tenant "customer-a": deployment_profile must be specified`,
				`tenant "customer-a": exactly one database must be defined`,
				`tenant "customer-a": exactly one radius server must be defined`,
				`tenant "customer-a": at least one nas assignment must be defined`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateTenant("customer-a", test.tenant, testTemplateLoader(t))
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateTenant() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateTenant() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateRADIUSServer(t *testing.T) {
	validServer := model.RADIUSServer{
		Version:            "3.2.10",
		AuthenticationPort: 1812,
		AccountingPort:     1813,
		COAPort:            3799,
	}

	tests := []struct {
		name     string
		server   model.RADIUSServer
		wantErrs []string
	}{
		{
			name:   "valid RADIUS Server",
			server: validServer,
		},
		{
			name: "version missing",
			server: model.RADIUSServer{
				AuthenticationPort: 1812,
				AccountingPort:     1813,
				COAPort:            3799,
			},
			wantErrs: []string{
				`tenant "customer-a": radius server version must be specified`,
			},
		},
		{
			name: "version unsupported",
			server: model.RADIUSServer{
				Version:            "3.3.0",
				AuthenticationPort: 1812,
				AccountingPort:     1813,
				COAPort:            3799,
			},
			wantErrs: []string{
				`tenant "customer-a": radius server version "3.3.0" is not supported`,
			},
		},
		{
			name: "minimum valid ports",
			server: model.RADIUSServer{
				Version:            "3.2.10",
				AuthenticationPort: 1,
				AccountingPort:     1,
				COAPort:            1,
			},
		},
		{
			name: "maximum valid ports",
			server: model.RADIUSServer{
				Version:            "3.2.10",
				AuthenticationPort: 65535,
				AccountingPort:     65535,
				COAPort:            65535,
			},
		},
		{
			name: "authentication port missing",
			server: model.RADIUSServer{
				Version:        validServer.Version,
				AccountingPort: validServer.AccountingPort,
				COAPort:        validServer.COAPort,
			},
			wantErrs: []string{
				`tenant "customer-a": radius server authentication_port must be between 1 and 65535`,
			},
		},
		{
			name: "authentication port above range",
			server: model.RADIUSServer{
				Version:            validServer.Version,
				AuthenticationPort: 65536,
				AccountingPort:     validServer.AccountingPort,
				COAPort:            validServer.COAPort,
			},
			wantErrs: []string{
				`tenant "customer-a": radius server authentication_port must be between 1 and 65535`,
			},
		},
		{
			name: "accounting port missing",
			server: model.RADIUSServer{
				Version:            validServer.Version,
				AuthenticationPort: validServer.AuthenticationPort,
				COAPort:            validServer.COAPort,
			},
			wantErrs: []string{
				`tenant "customer-a": radius server accounting_port must be between 1 and 65535`,
			},
		},
		{
			name: "accounting port above range",
			server: model.RADIUSServer{
				Version:            validServer.Version,
				AuthenticationPort: validServer.AuthenticationPort,
				AccountingPort:     65536,
				COAPort:            validServer.COAPort,
			},
			wantErrs: []string{
				`tenant "customer-a": radius server accounting_port must be between 1 and 65535`,
			},
		},
		{
			name: "CoA port missing",
			server: model.RADIUSServer{
				Version:            validServer.Version,
				AuthenticationPort: validServer.AuthenticationPort,
				AccountingPort:     validServer.AccountingPort,
			},
			wantErrs: []string{
				`tenant "customer-a": radius server coa_port must be between 1 and 65535`,
			},
		},
		{
			name: "CoA port above range",
			server: model.RADIUSServer{
				Version:            validServer.Version,
				AuthenticationPort: validServer.AuthenticationPort,
				AccountingPort:     validServer.AccountingPort,
				COAPort:            65536,
			},
			wantErrs: []string{
				`tenant "customer-a": radius server coa_port must be between 1 and 65535`,
			},
		},
		{
			name: "multiple invalid ports",
			server: model.RADIUSServer{
				Version:            validServer.Version,
				AuthenticationPort: 0,
				AccountingPort:     65536,
				COAPort:            -1,
			},
			wantErrs: []string{
				`tenant "customer-a": radius server authentication_port must be between 1 and 65535`,
				`tenant "customer-a": radius server accounting_port must be between 1 and 65535`,
				`tenant "customer-a": radius server coa_port must be between 1 and 65535`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateRADIUSServer("customer-a", test.server, testTemplateLoader(t))
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateRADIUSServer() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateRADIUSServer() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateDatabase(t *testing.T) {
	validExternalDatabase := model.Database{
		Engine:     "mysql",
		Deployment: "external",
		Host:       "db.example.com",
		Port:       3306,
		Database:   "radius",
		Username:   "radius",
		Password:   "secret",
	}

	validContainerDatabase := model.Database{
		Engine:     "mysql",
		Deployment: "container",
		Database:   "radius",
		Username:   "radius",
		Password:   "secret",
	}

	tests := []struct {
		name     string
		database model.Database
		wantErrs []string
	}{
		{
			name:     "valid external database",
			database: validExternalDatabase,
		},
		{
			name:     "valid container database",
			database: validContainerDatabase,
		},
		{
			name: "default deployment is container",
			database: model.Database{
				Engine:   "mysql",
				Database: "radius",
				Username: "radius",
				Password: "secret",
			},
		},
		{
			name: "minimum valid external port",
			database: model.Database{
				Engine:     validExternalDatabase.Engine,
				Deployment: validExternalDatabase.Deployment,
				Host:       validExternalDatabase.Host,
				Port:       1,
				Database:   validExternalDatabase.Database,
				Username:   validExternalDatabase.Username,
				Password:   validExternalDatabase.Password,
			},
		},
		{
			name: "maximum valid external port",
			database: model.Database{
				Engine:     validExternalDatabase.Engine,
				Deployment: validExternalDatabase.Deployment,
				Host:       validExternalDatabase.Host,
				Port:       65535,
				Database:   validExternalDatabase.Database,
				Username:   validExternalDatabase.Username,
				Password:   validExternalDatabase.Password,
			},
		},
		{
			name: "engine missing",
			database: model.Database{
				Deployment: "container",
				Database:   validContainerDatabase.Database,
				Username:   validContainerDatabase.Username,
				Password:   validContainerDatabase.Password,
			},
			wantErrs: []string{
				`tenant "customer-a": database engine must be specified`,
			},
		},
		{
			name: "engine unsupported",
			database: model.Database{
				Engine:     "postgresql",
				Deployment: "container",
				Database:   validContainerDatabase.Database,
				Username:   validContainerDatabase.Username,
				Password:   validContainerDatabase.Password,
			},
			wantErrs: []string{
				`tenant "customer-a": database engine "postgresql" is not supported`,
			},
		},
		{
			name: "deployment unsupported",
			database: model.Database{
				Engine:     "mysql",
				Deployment: "unsupported",
				Database:   validContainerDatabase.Database,
				Username:   validContainerDatabase.Username,
				Password:   validContainerDatabase.Password,
			},
			wantErrs: []string{
				`tenant "customer-a": database deployment "unsupported" is not supported`,
			},
		},
		{
			name: "external host missing",
			database: model.Database{
				Engine:     validExternalDatabase.Engine,
				Deployment: "external",
				Port:       validExternalDatabase.Port,
				Database:   validExternalDatabase.Database,
				Username:   validExternalDatabase.Username,
				Password:   validExternalDatabase.Password,
			},
			wantErrs: []string{
				`tenant "customer-a": database host must be specified for external deployment`,
			},
		},
		{
			name: "external port below range",
			database: model.Database{
				Engine:     validExternalDatabase.Engine,
				Deployment: "external",
				Host:       validExternalDatabase.Host,
				Port:       0,
				Database:   validExternalDatabase.Database,
				Username:   validExternalDatabase.Username,
				Password:   validExternalDatabase.Password,
			},
			wantErrs: []string{
				`tenant "customer-a": database port must be between 1 and 65535 for external deployment`,
			},
		},
		{
			name: "external port above range",
			database: model.Database{
				Engine:     validExternalDatabase.Engine,
				Deployment: "external",
				Host:       validExternalDatabase.Host,
				Port:       65536,
				Database:   validExternalDatabase.Database,
				Username:   validExternalDatabase.Username,
				Password:   validExternalDatabase.Password,
			},
			wantErrs: []string{
				`tenant "customer-a": database port must be between 1 and 65535 for external deployment`,
			},
		},
		{
			name: "container host is optional",
			database: model.Database{
				Engine:     validContainerDatabase.Engine,
				Deployment: "container",
				Database:   validContainerDatabase.Database,
				Username:   validContainerDatabase.Username,
				Password:   validContainerDatabase.Password,
			},
		},
		{
			name: "container port is optional",
			database: model.Database{
				Engine:     validContainerDatabase.Engine,
				Deployment: "container",
				Host:       "unused.example.com",
				Database:   validContainerDatabase.Database,
				Username:   validContainerDatabase.Username,
				Password:   validContainerDatabase.Password,
			},
		},
		{
			name: "valid proxysql database",
			database: model.Database{
				Engine:     "mysql",
				Deployment: "proxysql",
				Host:       "db.example.com",
				Port:       3306,
				Database:   "radius",
				Username:   "radius",
				Password:   "secret",
			},
		},
		{
			name: "proxysql host missing",
			database: model.Database{
				Engine:     "mysql",
				Deployment: "proxysql",
				Port:       3306,
				Database:   "radius",
				Username:   "radius",
				Password:   "secret",
			},
			wantErrs: []string{
				`tenant "customer-a": database host must be specified for proxysql deployment`,
			},
		},
		{
			name: "proxysql port below range",
			database: model.Database{
				Engine:     "mysql",
				Deployment: "proxysql",
				Host:       "db.example.com",
				Port:       0,
				Database:   "radius",
				Username:   "radius",
				Password:   "secret",
			},
			wantErrs: []string{
				`tenant "customer-a": database port must be between 1 and 65535 for proxysql deployment`,
			},
		},
		{
			name: "proxysql port above range",
			database: model.Database{
				Engine:     "mysql",
				Deployment: "proxysql",
				Host:       "db.example.com",
				Port:       65536,
				Database:   "radius",
				Username:   "radius",
				Password:   "secret",
			},
			wantErrs: []string{
				`tenant "customer-a": database port must be between 1 and 65535 for proxysql deployment`,
			},
		},
		{
			name: "database name missing",
			database: model.Database{
				Engine:     validContainerDatabase.Engine,
				Deployment: "container",
				Username:   validContainerDatabase.Username,
				Password:   validContainerDatabase.Password,
			},
			wantErrs: []string{
				`tenant "customer-a": database name must be specified`,
			},
		},
		{
			name: "username missing",
			database: model.Database{
				Engine:     validContainerDatabase.Engine,
				Deployment: "container",
				Database:   validContainerDatabase.Database,
				Password:   validContainerDatabase.Password,
			},
			wantErrs: []string{
				`tenant "customer-a": database username must be specified`,
			},
		},
		{
			name: "password missing",
			database: model.Database{
				Engine:     validContainerDatabase.Engine,
				Deployment: "container",
				Database:   validContainerDatabase.Database,
				Username:   validContainerDatabase.Username,
			},
			wantErrs: []string{
				`tenant "customer-a": database password must be specified`,
			},
		},
		{
			name: "multiple invalid properties",
			database: model.Database{
				Deployment: "external",
			},
			wantErrs: []string{
				`tenant "customer-a": database engine must be specified`,
				`tenant "customer-a": database host must be specified for external deployment`,
				`tenant "customer-a": database port must be between 1 and 65535 for external deployment`,
				`tenant "customer-a": database name must be specified`,
				`tenant "customer-a": database username must be specified`,
				`tenant "customer-a": database password must be specified`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateDatabase("customer-a", test.database)
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateDatabase() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateDatabase() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateNASDevice(t *testing.T) {
	tests := []struct {
		name     string
		device   model.NASDevice
		wantErrs []string
	}{
		{
			name:   "IPv4 address and vendor specified",
			device: model.NASDevice{IPAddress: "10.10.10.1", Vendor: "mikrotik"},
		},
		{
			name:   "IPv6 address and vendor specified",
			device: model.NASDevice{IPAddress: "2001:db8::1", Vendor: "mikrotik"},
		},
		{
			name:   "IP address missing",
			device: model.NASDevice{Vendor: "mikrotik"},
			wantErrs: []string{
				`nas device "core": ip_address must be a valid IPv4 or IPv6 address`,
			},
		},
		{
			name:   "IP address invalid",
			device: model.NASDevice{IPAddress: "not-an-ip", Vendor: "mikrotik"},
			wantErrs: []string{
				`nas device "core": ip_address must be a valid IPv4 or IPv6 address`,
			},
		},
		{
			name:   "vendor missing",
			device: model.NASDevice{IPAddress: "10.10.10.1"},
			wantErrs: []string{
				`nas device "core": vendor must be specified`,
			},
		},
		{
			name:   "IP address and vendor missing",
			device: model.NASDevice{},
			wantErrs: []string{
				`nas device "core": ip_address must be a valid IPv4 or IPv6 address`,
				`nas device "core": vendor must be specified`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateNASDevice("core", test.device)
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateNASDevice() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateNASDevice() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateTrustedRADIUSClient(t *testing.T) {
	tests := []struct {
		name     string
		client   model.TrustedRADIUSClient
		wantErrs []string
	}{
		{
			name:   "IPv4 address specified",
			client: model.TrustedRADIUSClient{IPAddress: "10.10.10.1"},
		},
		{
			name:   "IPv6 address specified",
			client: model.TrustedRADIUSClient{IPAddress: "2001:db8::1"},
		},
		{
			name:   "IP address missing",
			client: model.TrustedRADIUSClient{},
			wantErrs: []string{
				`trusted radius client "monitoring": ip_address must be a valid IPv4 or IPv6 address`,
			},
		},
		{
			name:   "IP address invalid",
			client: model.TrustedRADIUSClient{IPAddress: "not-an-ip"},
			wantErrs: []string{
				`trusted radius client "monitoring": ip_address must be a valid IPv4 or IPv6 address`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateTrustedRADIUSClient("monitoring", test.client)
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateTrustedRADIUSClient() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateTrustedRADIUSClient() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateTrustedRADIUSClientAssignment(t *testing.T) {
	tests := []struct {
		name       string
		assignment model.TrustedRADIUSClientAssignment
		wantErrs   []string
	}{
		{
			name: "valid assignment",
			assignment: model.TrustedRADIUSClientAssignment{
				CredentialProfile: "default",
			},
		},
		{
			name:       "credential profile missing",
			assignment: model.TrustedRADIUSClientAssignment{},
			wantErrs: []string{
				`tenant "customer-a": trusted radius client assignment "monitoring": credential_profile must be specified`,
			},
		},
		{
			name: "all properties missing",
			wantErrs: []string{
				`tenant "customer-a": trusted radius client assignment "monitoring": credential_profile must be specified`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateTrustedRADIUSClientAssignment("customer-a", "monitoring", test.assignment)
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateTrustedRADIUSClientAssignment() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateTrustedRADIUSClientAssignment() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateTrustedRADIUSClientAssignmentReferences(t *testing.T) {
	globalObjects := model.GlobalObjects{
		CredentialProfiles: map[string]model.CredentialProfile{"default": {}},
		TrustedRADIUSClients: map[string]model.TrustedRADIUSClient{
			"monitoring": {},
		},
	}

	tests := []struct {
		name       string
		assignment model.TrustedRADIUSClientAssignment
		wantErrs   []string
	}{
		{
			name: "valid references",
			assignment: model.TrustedRADIUSClientAssignment{
				CredentialProfile: "default",
			},
		},
		{
			name: "credential profile missing",
			assignment: model.TrustedRADIUSClientAssignment{
				CredentialProfile: "missing-credentials",
			},
			wantErrs: []string{
				`tenant "customer-a": trusted radius client assignment "monitoring": credential profile "missing-credentials" does not exist`,
			},
		},
		{
			name: "all references missing",
			assignment: model.TrustedRADIUSClientAssignment{
				CredentialProfile: "missing-credentials",
			},
			wantErrs: []string{
				`tenant "customer-a": trusted radius client assignment "monitoring": credential profile "missing-credentials" does not exist`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validationErrors := validateTrustedRADIUSClientAssignmentReferences("customer-a", "monitoring", test.assignment, globalObjects)
			if len(validationErrors) != len(test.wantErrs) {
				t.Fatalf("validateTrustedRADIUSClientAssignmentReferences() returned %d errors, want %d", len(validationErrors), len(test.wantErrs))
			}

			for index, wantErr := range test.wantErrs {
				if got := validationErrors[index].Error(); got != wantErr {
					t.Errorf("validateTrustedRADIUSClientAssignmentReferences() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateTrustedRADIUSClientAssignmentRelationships(t *testing.T) {
	globalObjects := model.GlobalObjects{
		TrustedRADIUSClients: map[string]model.TrustedRADIUSClient{
			"monitoring":   {},
			"provisioning": {},
		},
	}
	tenant := model.Tenant{TrustedRADIUSClientAssignments: map[string]model.TrustedRADIUSClientAssignment{
		"monitoring":   {},
		"provisioning": {},
	}}

	validationErrors := validateTenantRelationships("customer-a", tenant, globalObjects)
	if len(validationErrors) != 0 {
		t.Fatalf("validateTenantRelationships() returned %v, want no errors", validationErrors)
	}
}

func TestValidateCredentialProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile model.CredentialProfile
		wantErr string
	}{
		{
			name:    "shared secret specified",
			profile: model.CredentialProfile{SharedSecret: "secret"},
		},
		{
			name:    "shared secret missing",
			profile: model.CredentialProfile{},
			wantErr: `credential profile "default": shared_secret must be specified`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validateCredentialProfile("default", test.profile)
			if len(errs) == 0 {
				if test.wantErr != "" {
					t.Fatal("validateCredentialProfile() returned no errors")
				}
				return
			}

			if test.wantErr == "" {
				t.Fatalf("validateCredentialProfile() error = %v, want none", errs[0])
			}
			if got := errs[0].Error(); got != test.wantErr {
				t.Fatalf("validateCredentialProfile() error = %q, want %q", got, test.wantErr)
			}
		})
	}
}

func TestSortedKeys(t *testing.T) {
	values := map[string]int{
		"zulu":   1,
		"alpha":  2,
		"middle": 3,
	}

	if got, want := sortedKeys(values), []string{"alpha", "middle", "zulu"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedKeys() = %v, want %v", got, want)
	}
}

func TestValidateAccountingProfileStaleSessionTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout string
		wantErr string
	}{
		{name: "omitted"},
		{name: "valid minutes", timeout: "20m"},
		{name: "valid hour", timeout: "1h"},
		{name: "invalid duration", timeout: "twenty minutes", wantErr: `accounting profile "default": stale_session_timeout must be a valid duration`},
		{name: "zero duration", timeout: "0s", wantErr: `accounting profile "default": stale_session_timeout must be greater than zero`},
		{name: "negative duration", timeout: "-5m", wantErr: `accounting profile "default": stale_session_timeout must be greater than zero`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validateAccountingProfile("default", model.AccountingProfile{StaleSessionTimeout: test.timeout})
			if test.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("validateAccountingProfile() errors = %v, want none", errs)
				}
				return
			}
			if len(errs) != 1 {
				t.Fatalf("validateAccountingProfile() returned %d errors, want 1", len(errs))
			}
			if got := errs[0].Error(); got != test.wantErr {
				t.Fatalf("validateAccountingProfile() error = %q, want %q", got, test.wantErr)
			}
		})
	}
}

func TestValidateAuthenticationProfile(t *testing.T) {
	one := 1
	zero := 0
	minusOne := -1
	tests := []struct {
		name    string
		profile model.AuthenticationProfile
		wantErr string
	}{
		{name: "valid", profile: model.AuthenticationProfile{SimultaneousUse: &one}},
		{
			name:    "unspecified",
			profile: model.AuthenticationProfile{},
		},
		{
			name: "zero",
			profile: model.AuthenticationProfile{
				SimultaneousUse: &zero,
			}, wantErr: `authentication profile "default": simultaneous_use must be greater than zero`},
		{name: "negative", profile: model.AuthenticationProfile{SimultaneousUse: &minusOne}, wantErr: `authentication profile "default": simultaneous_use must be greater than zero`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validateAuthenticationProfile("default", test.profile)
			if test.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("errors = %v, want none", errs)
				}
				return
			}
			if len(errs) != 1 || errs[0].Error() != test.wantErr {
				t.Fatalf("errors = %v, want %q", errs, test.wantErr)
			}
		})
	}
}

func TestValidateTenantAuthenticationProfileReference(t *testing.T) {
	one := 1
	globals := model.GlobalObjects{AuthenticationProfiles: map[string]model.AuthenticationProfile{"default": {SimultaneousUse: &one}}}
	tenant := model.Tenant{AuthenticationProfile: "default"}
	if errs := validateTenantReferences("customer-a", tenant, globals); len(errs) != 0 {
		t.Fatalf("errors = %v, want none", errs)
	}
	tenant.AuthenticationProfile = "missing"
	errs := validateTenantReferences("customer-a", tenant, globals)
	if len(errs) != 1 || errs[0].Error() != `tenant "customer-a": authentication profile "missing" does not exist` {
		t.Fatalf("errors = %v", errs)
	}
}

func TestValidateDeploymentProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile model.DeploymentProfile
		wantErr string
	}{
		{
			name: "valid",
			profile: model.DeploymentProfile{
				Template: "default",
			},
		},
		{
			name:    "template unspecified",
			profile: model.DeploymentProfile{},
			wantErr: `deployment profile "default": template must be specified`,
		},
		{
			name: "valid with overlays",
			profile: model.DeploymentProfile{
				Template: "default",
				Overlays: []string{
					"coa-relay-test",
					"debug-logging",
				},
			},
		},
		{
			name: "current directory template",
			profile: model.DeploymentProfile{
				Template: ".",
			},
			wantErr: `deployment profile "default": template "." is invalid`,
		},
		{
			name: "invalid overlay",
			profile: model.DeploymentProfile{
				Template: "default",
				Overlays: []string{
					"../something",
				},
			},
			wantErr: `deployment profile "default": overlay "../something" is invalid`,
		},
		{
			name: "current directory overlay",
			profile: model.DeploymentProfile{
				Template: "default",
				Overlays: []string{"."},
			},
			wantErr: `deployment profile "default": overlay "." is invalid`,
		},
		{
			name: "overlay containing path separator",
			profile: model.DeploymentProfile{
				Template: "default",
				Overlays: []string{
					"experiments/coa-relay",
				},
			},
			wantErr: `deployment profile "default": overlay "experiments/coa-relay" is invalid`,
		},
		{
			name: "valid remove path",
			profile: model.DeploymentProfile{
				Template: "default",
				Remove: []string{
					"sites-enabled/inner-tunnel",
					"mods-enabled/sql",
					"users",
					"mods-config/files/authorize",
				},
			},
		},
		{
			name: "remove path with backslash",
			profile: model.DeploymentProfile{
				Template: "default",
				Remove: []string{
					`sites-enabled\inner-tunnel`,
				},
			},
			wantErr: `deployment profile "default": remove[0]: path must use '/' as the separator`,
		},
		{
			name: "absolute remove path",
			profile: model.DeploymentProfile{
				Template: "default",
				Remove: []string{
					"/etc/freeradius/users",
				},
			},
			wantErr: `deployment profile "default": remove[0]: path must be a valid relative path`,
		},
		{
			name: "parent traversal remove path",
			profile: model.DeploymentProfile{
				Template: "default",
				Remove: []string{
					"../users",
				},
			},
			wantErr: `deployment profile "default": remove[0]: path must be a valid relative path`,
		},
		{
			name: "nested parent traversal remove path",
			profile: model.DeploymentProfile{
				Template: "default",
				Remove: []string{
					"sites-enabled/../../users",
				},
			},
			wantErr: `deployment profile "default": remove[0]: path must be a valid relative path`,
		},
		{
			name: "empty remove path",
			profile: model.DeploymentProfile{
				Template: "default",
				Remove: []string{
					"",
				},
			},
			wantErr: `deployment profile "default": remove[0]: path must not be empty`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validateDeploymentProfile("default", test.profile)
			if test.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("errors = %v, want none", errs)
				}
				return
			}
			if len(errs) != 1 || errs[0].Error() != test.wantErr {
				t.Fatalf("errors = %v, want %q", errs, test.wantErr)
			}
		})
	}
}

func TestValidateTenantDeploymentProfileReference(t *testing.T) {
	globals := model.GlobalObjects{DeploymentProfiles: map[string]model.DeploymentProfile{"default": {Template: "default"}}}
	tenant := model.Tenant{DeploymentProfile: "default"}
	if errs := validateTenantReferences("customer-a", tenant, globals); len(errs) != 0 {
		t.Fatalf("errors = %v, want none", errs)
	}
	tenant.DeploymentProfile = "missing"
	errs := validateTenantReferences("customer-a", tenant, globals)
	if len(errs) != 1 || errs[0].Error() != `tenant "customer-a": deployment profile "missing" does not exist` {
		t.Fatalf("errors = %v", errs)
	}
}

func TestValidateTemplateAvailability(t *testing.T) {
	configuration := model.Configuration{
		GlobalObjects: model.GlobalObjects{
			DeploymentProfiles: map[string]model.DeploymentProfile{
				"default": {Template: "default"},
			},
		},
		Tenants: map[string]model.Tenant{
			"customer-a": {
				DeploymentProfile: "default",
				RADIUSServer:      model.RADIUSServer{Version: "3.2.10"},
			},
		},
	}

	tests := []struct {
		name     string
		mutate   func(*model.Configuration)
		wantErrs []string
	}{
		{
			name: "available template set and overlays",
			mutate: func(configuration *model.Configuration) {
				profile := configuration.GlobalObjects.DeploymentProfiles["default"]
				profile.Overlays = []string{"test-overlay"}
				configuration.GlobalObjects.DeploymentProfiles["default"] = profile
			},
		},
		{
			name: "missing template set",
			mutate: func(configuration *model.Configuration) {
				profile := configuration.GlobalObjects.DeploymentProfiles["default"]
				profile.Template = "missing"
				configuration.GlobalObjects.DeploymentProfiles["default"] = profile
			},
			wantErrs: []string{`tenant "customer-a": deployment profile "default": template set "missing" is not available for FreeRADIUS version "3.2.10"`},
		},
		{
			name: "missing overlay",
			mutate: func(configuration *model.Configuration) {
				profile := configuration.GlobalObjects.DeploymentProfiles["default"]
				profile.Overlays = []string{"missing"}
				configuration.GlobalObjects.DeploymentProfiles["default"] = profile
			},
			wantErrs: []string{`tenant "customer-a": deployment profile "default": overlay "missing" is not available for FreeRADIUS version "3.2.10"`},
		},
		{
			name: "multiple missing overlays",
			mutate: func(configuration *model.Configuration) {
				profile := configuration.GlobalObjects.DeploymentProfiles["default"]
				profile.Overlays = []string{"first", "second"}
				configuration.GlobalObjects.DeploymentProfiles["default"] = profile
			},
			wantErrs: []string{
				`tenant "customer-a": deployment profile "default": overlay "first" is not available for FreeRADIUS version "3.2.10"`,
				`tenant "customer-a": deployment profile "default": overlay "second" is not available for FreeRADIUS version "3.2.10"`,
			},
		},
		{
			name: "missing version",
			mutate: func(configuration *model.Configuration) {
				tenant := configuration.Tenants["customer-a"]
				tenant.RADIUSServer.Version = ""
				configuration.Tenants["customer-a"] = tenant
			},
		},
		{
			name: "unsupported version",
			mutate: func(configuration *model.Configuration) {
				tenant := configuration.Tenants["customer-a"]
				tenant.RADIUSServer.Version = "unsupported"
				configuration.Tenants["customer-a"] = tenant
			},
		},
		{
			name: "missing deployment profile reference",
			mutate: func(configuration *model.Configuration) {
				tenant := configuration.Tenants["customer-a"]
				tenant.DeploymentProfile = "missing"
				configuration.Tenants["customer-a"] = tenant
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configurationCopy := configuration
			configurationCopy.GlobalObjects.DeploymentProfiles = map[string]model.DeploymentProfile{}
			for identifier, profile := range configuration.GlobalObjects.DeploymentProfiles {
				configurationCopy.GlobalObjects.DeploymentProfiles[identifier] = profile
			}
			configurationCopy.Tenants = map[string]model.Tenant{}
			for identifier, tenant := range configuration.Tenants {
				configurationCopy.Tenants[identifier] = tenant
			}
			test.mutate(&configurationCopy)

			errs := validateTemplateAvailability(configurationCopy, testTemplateLoader(t))
			if len(errs) != len(test.wantErrs) {
				t.Fatalf("validateTemplateAvailability() returned %d errors, want %d: %v", len(errs), len(test.wantErrs), errs)
			}
			for index, wantErr := range test.wantErrs {
				if got := errs[index].Error(); got != wantErr {
					t.Errorf("validateTemplateAvailability() error = %q, want %q", got, wantErr)
				}
			}
		})
	}
}

func TestValidateIncludesTemplateAvailability(t *testing.T) {
	configuration := model.Configuration{
		GlobalObjects: model.GlobalObjects{
			DeploymentProfiles: map[string]model.DeploymentProfile{
				"default": {Template: "missing"},
			},
		},
		Tenants: map[string]model.Tenant{
			"customer-a": {
				DeploymentProfile: "default",
				RADIUSServer:      model.RADIUSServer{Version: "3.2.10"},
			},
		},
	}

	err := Validate(configuration, testTemplateLoader(t))
	if err == nil || !strings.Contains(err.Error(), `tenant "customer-a": deployment profile "default": template set "missing" is not available for FreeRADIUS version "3.2.10"`) {
		t.Fatalf("Validate() error = %v, want template availability error", err)
	}
}
