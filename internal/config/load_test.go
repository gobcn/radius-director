package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gobcn/radius-director/internal/model"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := []byte(`global_objects:
  credential_profiles:
    default:
      shared_secret: secret
  authentication_profiles:
    default:
      simultaneous_use: 1
  accounting_profiles:
    default:
      stale_session_timeout: 20m
  deployment_profiles:
    default:
      template: default
      overlays:
        - coa-relay-test
        - debug-logging
  nas_devices:
    core:
      ip_address: 10.10.10.1
      vendor: mikrotik
  trusted_radius_clients:
    monitoring:
      ip_address: 10.10.10.2
tenants:
  customer-a:
    authentication_profile: default
    deployment_profile: default
    database:
      engine: mysql
      host: db.example.com
      port: 3306
      database: radius
      username: radius
      password: secret
    radius_server:
      version: 3.2.10
      authentication_port: 1812
      accounting_port: 1813
      coa_port: 3799
    nas_assignments:
      core:
        credential_profile: default
        accounting_profile: default
        monitoring_profile: default
        require_message_authenticator: auto
      edge:
        credential_profile: default
        accounting_profile: default
        monitoring_profile: default
    trusted_radius_client_assignments:
      monitoring:
        credential_profile: default
        require_message_authenticator: no
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	configuration, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := configuration.GlobalObjects.CredentialProfiles["default"].SharedSecret; got != "secret" {
		t.Fatalf("credential profile shared secret = %q, want %q", got, "secret")
	}
	got := configuration.GlobalObjects.AuthenticationProfiles["default"].SimultaneousUse
	if got == nil || *got != 1 {
		t.Fatalf("authentication profile simultaneous use = %d, want 1", got)
	}
	if got := configuration.Tenants["customer-a"].AuthenticationProfile; got != "default" {
		t.Fatalf("tenant authentication profile = %q, want %q", got, "default")
	}
	if got := configuration.Tenants["customer-a"].DeploymentProfile; got != "default" {
		t.Fatalf("tenant deployment profile = %q, want %q", got, "default")
	}
	deploymentProfile := configuration.GlobalObjects.DeploymentProfiles["default"]
	if got := len(deploymentProfile.Overlays); got != 2 {
		t.Fatalf("deployment profile overlay count = %d, want %d", got, 2)
	}
	if got := deploymentProfile.Overlays[0]; got != "coa-relay-test" {
		t.Fatalf("first deployment profile overlay = %q, want %q", got, "coa-relay-test")
	}
	if got := deploymentProfile.Overlays[1]; got != "debug-logging" {
		t.Fatalf("second deployment profile overlay = %q, want %q", got, "debug-logging")
	}
	if got := configuration.GlobalObjects.AccountingProfiles["default"].StaleSessionTimeout; got != "20m" {
		t.Fatalf("accounting profile stale session timeout = %q, want %q", got, "20m")
	}
	if got := configuration.Tenants["customer-a"].NASAssignments["core"].RequireMessageAuthenticator; got == nil || *got != model.RequireMessageAuthenticatorAuto {
		t.Fatalf("NAS assignment require_message_authenticator = %v, want auto", got)
	}
	if got := configuration.Tenants["customer-a"].NASAssignments["edge"].RequireMessageAuthenticator; got != nil {
		t.Fatalf("omitted NAS assignment require_message_authenticator = %v, want nil", got)
	}
	if got := configuration.GlobalObjects.TrustedRADIUSClients["monitoring"].IPAddress; got != "10.10.10.2" {
		t.Fatalf("trusted RADIUS client IP address = %q, want %q", got, "10.10.10.2")
	}
	if got := configuration.Tenants["customer-a"].TrustedRADIUSClientAssignments["monitoring"].RequireMessageAuthenticator; got == nil || *got != model.RequireMessageAuthenticatorNo {
		t.Fatalf("trusted RADIUS client assignment require_message_authenticator = %v, want no", got)
	}
	if got := configuration.Tenants["customer-a"].RADIUSServer.AuthenticationPort; got != 1812 {
		t.Fatalf("RADIUS Server authentication port = %d, want 1812", got)
	}
	if got := configuration.Tenants["customer-a"].RADIUSServer.Version; got != "3.2.10" {
		t.Fatalf("RADIUS Server version = %q, want %q", got, "3.2.10")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(path, []byte("global_objects: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want YAML parsing error")
	}
}

func TestLoadRejectsDuplicateAssignmentIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		contents string
	}{
		{
			name: "NAS assignment",
			contents: `tenants:
  customer-a:
    nas_assignments:
      core:
        credential_profile: default
      core:
        credential_profile: alternate
`,
		},
		{
			name: "trusted RADIUS client assignment",
			contents: `tenants:
  customer-a:
    trusted_radius_client_assignments:
      sonar:
        credential_profile: default
      sonar:
        credential_profile: alternate
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("Load() error = nil, want duplicate assignment identifier error")
			}
		})
	}
}
