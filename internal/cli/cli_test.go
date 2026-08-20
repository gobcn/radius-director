package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gobcn/radius-director/internal/deployment/docker"
	"github.com/gobcn/radius-director/internal/schemas"
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

func testSchemaLoader(t *testing.T) schemas.Loader {
	t.Helper()

	directory, err := filepath.Abs(filepath.Join("..", "..", "schemas"))
	if err != nil {
		t.Fatalf("resolve schema directory: %v", err)
	}

	return schemas.NewLoader(os.DirFS(directory))
}

func exampleConfigPath(t *testing.T) string {
	t.Helper()

	path, err := filepath.Abs(filepath.Join("..", "..", "resources", "example.yaml"))
	if err != nil {
		t.Fatalf("resolve example configuration: %v", err)
	}

	return path
}

func createTestAssets(t *testing.T, directory string) {
	t.Helper()

	templatesDirectory := filepath.Join(directory, "templates")
	schemasDirectory := filepath.Join(directory, "schemas")

	if err := os.MkdirAll(templatesDirectory, 0o755); err != nil {
		t.Fatalf("create templates directory: %v", err)
	}

	if err := os.MkdirAll(schemasDirectory, 0o755); err != nil {
		t.Fatalf("create schemas directory: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(templatesDirectory, "default.conf"),
		[]byte("template content"),
		0o644,
	); err != nil {
		t.Fatalf("write test template: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(schemasDirectory, "schema.sql"),
		[]byte("schema content"),
		0o644,
	); err != nil {
		t.Fatalf("write test schema: %v", err)
	}
}

type fakeRuntimeInitializer struct {
	calls       int
	root        string
	networkName string
	err         error
}

func (f *fakeRuntimeInitializer) Init(
	_ context.Context,
	root string,
	networkName string,
) error {
	f.calls++
	f.root = root
	f.networkName = networkName

	return f.err
}

func TestRunInit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fake := &fakeRuntimeInitializer{}

	exitCode := Run(
		[]string{
			"init",
			"/runtime/test",
			"radius-director-test",
		},
		&stdout,
		&stderr,
		testTemplateLoader(t),
		testSchemaLoader(t),
		fake,
	)

	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}

	if fake.calls != 1 {
		t.Fatalf("Init calls = %d, want 1", fake.calls)
	}

	if fake.root != "/runtime/test" {
		t.Fatalf("root = %q, want %q", fake.root, "/runtime/test")
	}

	if fake.networkName != "radius-director-test" {
		t.Fatalf(
			"network name = %q, want %q",
			fake.networkName,
			"radius-director-test",
		)
	}

	if !strings.Contains(
		stdout.String(),
		"RADIUS Director runtime initialized successfully",
	) {
		t.Fatalf("stdout = %q, want success message", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := Run([]string{"--help"}, &stdout, &stderr, testTemplateLoader(t), testSchemaLoader(t), nil); exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("help output = %q, want usage information", stdout.String())
	}
	if count := strings.Count(stdout.String(), "Usage:"); count != 1 {
		t.Fatalf("help output contains %d usage sections, want 1", count)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunInitHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fake := &fakeRuntimeInitializer{}

	exitCode := Run(
		[]string{"init", "--help"},
		&stdout,
		&stderr,
		testTemplateLoader(t),
		testSchemaLoader(t),
		fake,
	)

	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(
		stdout.String(),
		"radius-director init <runtime-directory> <network-name>",
	) {
		t.Fatalf("stdout = %q, want init usage", stdout.String())
	}

	if fake.calls != 0 {
		t.Fatalf("Init calls = %d, want 0", fake.calls)
	}
}

func TestRunInitRejectsWrongArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "none",
			args: []string{"init"},
		},
		{
			name: "one",
			args: []string{"init", "/runtime/test"},
		},
		{
			name: "three",
			args: []string{"init", "/runtime/test", "radius-test", "extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			fake := &fakeRuntimeInitializer{}

			exitCode := Run(
				tt.args,
				&stdout,
				&stderr,
				testTemplateLoader(t),
				testSchemaLoader(t),
				fake,
			)

			if exitCode != 2 {
				t.Fatalf(
					"Run() exit code = %d, want 2; stderr = %q",
					exitCode,
					stderr.String(),
				)
			}

			if fake.calls != 0 {
				t.Fatalf("Init calls = %d, want 0", fake.calls)
			}

			if !strings.Contains(
				stderr.String(),
				"init requires a runtime directory and network name",
			) {
				t.Fatalf("stderr = %q, want argument error", stderr.String())
			}
		})
	}
}

func TestRunInitReportsInitializationError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fake := &fakeRuntimeInitializer{
		err: errors.New("test initialization failure"),
	}

	exitCode := Run(
		[]string{
			"init",
			"/runtime/test",
			"radius-director-test",
		},
		&stdout,
		&stderr,
		testTemplateLoader(t),
		testSchemaLoader(t),
		fake,
	)

	if exitCode != 1 {
		t.Fatalf(
			"Run() exit code = %d, want 1; stderr = %q",
			exitCode,
			stderr.String(),
		)
	}

	if fake.calls != 1 {
		t.Fatalf("Init calls = %d, want 1", fake.calls)
	}

	if !strings.Contains(stderr.String(), "test initialization failure") {
		t.Fatalf(
			"stderr = %q, want initialization error",
			stderr.String(),
		)
	}
}

func TestRunExportAssets(t *testing.T) {
	sourceDirectory := t.TempDir()
	createTestAssets(t, sourceDirectory)

	destinationDirectory := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := runExportAssetsFromRoot(
		[]string{destinationDirectory},
		&stdout,
		&stderr,
		sourceDirectory,
	); exitCode != 0 {
		t.Fatalf(
			"runExportAssetsFromRoot() exit code = %d, want 0; stderr = %q",
			exitCode,
			stderr.String(),
		)
	}

	if !strings.Contains(
		stdout.String(),
		"exported successfully",
	) {
		t.Fatalf(
			"stdout = %q, want successful export message",
			stdout.String(),
		)
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	expectedFiles := map[string]string{
		filepath.Join("templates", "default.conf"): "template content",
		filepath.Join("schemas", "schema.sql"):     "schema content",
	}

	for relativePath, expectedContent := range expectedFiles {
		path := filepath.Join(destinationDirectory, relativePath)

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf(
				"read exported file %q: %v",
				relativePath,
				err,
			)
		}

		if string(content) != expectedContent {
			t.Errorf(
				"exported file %q = %q, want %q",
				relativePath,
				string(content),
				expectedContent,
			)
		}
	}
}

func TestRunExportAssetsRequiresDestination(t *testing.T) {
	sourceDirectory := t.TempDir()
	createTestAssets(t, sourceDirectory)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := runExportAssetsFromRoot(
		nil,
		&stdout,
		&stderr,
		sourceDirectory,
	); exitCode != 2 {
		t.Fatalf(
			"runExportAssetsFromRoot() exit code = %d, want 2",
			exitCode,
		)
	}

	if !strings.Contains(
		stderr.String(),
		"export assets requires an output directory",
	) {
		t.Fatalf(
			"stderr = %q, want missing-destination error",
			stderr.String(),
		)
	}
}

func TestRunExportAssetsRequiresExactlyOneDestination(t *testing.T) {
	sourceDirectory := t.TempDir()
	createTestAssets(t, sourceDirectory)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := runExportAssetsFromRoot(
		[]string{t.TempDir(), t.TempDir()},
		&stdout,
		&stderr,
		sourceDirectory,
	); exitCode != 2 {
		t.Fatalf(
			"runExportAssetsFromRoot() exit code = %d, want 2",
			exitCode,
		)
	}

	if !strings.Contains(
		stderr.String(),
		"export assets requires an output directory",
	) {
		t.Fatalf(
			"stderr = %q, want missing-destination error",
			stderr.String(),
		)
	}
}

func TestRunExportAssetsFromRootUsesSpecifiedFactoryLibrary(t *testing.T) {
	factoryDirectory := t.TempDir()
	adminDirectory := t.TempDir()
	destinationDirectory := t.TempDir()

	createTestAssets(t, factoryDirectory)
	createTestAssets(t, adminDirectory)

	if err := os.WriteFile(
		filepath.Join(factoryDirectory, "templates", "default.conf"),
		[]byte("factory template"),
		0o644,
	); err != nil {
		t.Fatalf("write factory template: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(adminDirectory, "templates", "default.conf"),
		[]byte("administrator template"),
		0o644,
	); err != nil {
		t.Fatalf("write administrator template: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(adminDirectory, "templates", "administrator-only.conf"),
		[]byte("administrator-only"),
		0o644,
	); err != nil {
		t.Fatalf("write administrator-only template: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := runExportAssetsFromRoot(
		[]string{destinationDirectory},
		&stdout,
		&stderr,
		factoryDirectory,
	); exitCode != 0 {
		t.Fatalf(
			"runExportAssetsFromRoot() exit code = %d, want 0; stderr = %q",
			exitCode,
			stderr.String(),
		)
	}

	if _, err := os.Stat(
		filepath.Join(destinationDirectory, "templates", "administrator-only.conf"),
	); !os.IsNotExist(err) {
		t.Fatalf(
			"administrator-only template was exported; err = %v",
			err,
		)
	}

	content, err := os.ReadFile(
		filepath.Join(destinationDirectory, "templates", "default.conf"),
	)
	if err != nil {
		t.Fatalf("read exported template: %v", err)
	}

	if string(content) != "factory template" {
		t.Fatalf(
			"exported template = %q, want factory template",
			string(content),
		)
	}
}

func TestRunExportAssetsRefusesExistingAssets(t *testing.T) {
	sourceDirectory := t.TempDir()
	createTestAssets(t, sourceDirectory)

	destinationDirectory := t.TempDir()

	existingTemplates := filepath.Join(destinationDirectory, "templates")

	if err := os.MkdirAll(existingTemplates, 0o755); err != nil {
		t.Fatalf("create existing templates directory: %v", err)
	}

	customFile := filepath.Join(existingTemplates, "custom.conf")
	const customContent = "administrator customization"

	if err := os.WriteFile(
		customFile,
		[]byte(customContent),
		0o644,
	); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := runExportAssetsFromRoot(
		[]string{destinationDirectory},
		&stdout,
		&stderr,
		sourceDirectory,
	); exitCode != 1 {
		t.Fatalf(
			"runExportAssetsFromRoot() exit code = %d, want 1",
			exitCode,
		)
	}

	if !strings.Contains(
		stderr.String(),
		"refusing to overwrite",
	) {
		t.Fatalf(
			"stderr = %q, want refusal-to-overwrite error",
			stderr.String(),
		)
	}

	content, err := os.ReadFile(customFile)
	if err != nil {
		t.Fatalf("read custom template: %v", err)
	}

	if string(content) != customContent {
		t.Errorf(
			"custom template was modified: got %q, want %q",
			string(content),
			customContent,
		)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunExportAssetsExplicitDirectoryOverridesConfiguredDirectory(t *testing.T) {
	sourceDirectory := t.TempDir()
	createTestAssets(t, sourceDirectory)

	configuredDirectory := t.TempDir()
	explicitDirectory := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := runExportAssetsFromRoot(
		[]string{explicitDirectory},
		&stdout,
		&stderr,
		sourceDirectory,
	); exitCode != 0 {
		t.Fatalf(
			"runExportAssetsFromRoot() exit code = %d, want 0; stderr = %q",
			exitCode,
			stderr.String(),
		)
	}

	if _, err := os.Stat(filepath.Join(
		explicitDirectory,
		"templates",
		"default.conf",
	)); err != nil {
		t.Fatalf("expected export in explicit directory: %v", err)
	}

	if _, err := os.Stat(filepath.Join(
		configuredDirectory,
		"templates",
		"default.conf",
	)); !os.IsNotExist(err) {
		t.Fatalf(
			"configured directory was unexpectedly populated; err = %v",
			err,
		)
	}
}

func TestRunExportHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := Run(
		[]string{"export", "assets", "--help"},
		&stdout,
		&stderr,
		testTemplateLoader(t),
		testSchemaLoader(t),
		nil,
	); exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(
		stdout.String(),
		"export assets <output-directory>",
	) {
		t.Fatalf(
			"stdout = %q, want export assets usage",
			stdout.String(),
		)
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("global_objects: {}\ntenants: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := Run([]string{"validate", path}, &stdout, &stderr, testTemplateLoader(t), testSchemaLoader(t), nil); exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}
	if got := stdout.String(); got != "Configuration parsed and validated successfully.\n" {
		t.Fatalf("stdout = %q, want success message", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunGenerate(t *testing.T) {
	configPath := exampleConfigPath(t)
	outputDir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := Run(
		[]string{"generate", configPath, outputDir},
		&stdout,
		&stderr,
		testTemplateLoader(t),
		testSchemaLoader(t),
		nil,
	); exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}

	if stdout.String() != "Configuration generated successfully.\n" {
		t.Fatalf("stdout = %q, want success message", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	tenantRoot := filepath.Join(outputDir, "customer-a")

	expectedFiles := []string{
		"clients.conf",
		"clients.d/radius-director.conf",
		"mods-available/sql",
		"mods-config/files/authorize",
		"mods-config/files/authorize.d/radius-director",
		"proxy.conf",
		"proxy.d/radius-director.conf",
		"sites-available/coa",
		"sites-available/default",
	}

	for _, relativePath := range expectedFiles {
		path := filepath.Join(tenantRoot, filepath.FromSlash(relativePath))

		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected generated file %q: %v", relativePath, err)
			continue
		}

		if !info.Mode().IsRegular() {
			t.Errorf("generated path %q is not a regular file", relativePath)
		}
	}

	expectedSymlinks := map[string]string{
		"mods-enabled/sql":      filepath.Join("..", "mods-available", "sql"),
		"sites-enabled/coa":     filepath.Join("..", "sites-available", "coa"),
		"sites-enabled/default": filepath.Join("..", "sites-available", "default"),
		"users":                 filepath.Join("mods-config", "files", "authorize"),
	}

	for relativePath, wantTarget := range expectedSymlinks {
		path := filepath.Join(tenantRoot, filepath.FromSlash(relativePath))

		target, err := os.Readlink(path)
		if err != nil {
			t.Errorf("expected generated symlink %q: %v", relativePath, err)
			continue
		}

		if target != wantTarget {
			t.Errorf("symlink %q = %q, want %q", relativePath, target, wantTarget)
		}
	}

	composePath := filepath.Join(outputDir, docker.ComposeOutputPath)

	composeContent, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("expected generated Docker Compose file: %v", err)
	}

	compose := string(composeContent)

	expectedComposeContent := []string{
		"radius-customer-a:",
		"image: freeradius/freeradius-server:3.2.10",
	}

	for _, value := range expectedComposeContent {
		if !strings.Contains(compose, value) {
			t.Errorf("generated Docker Compose file does not contain %q:\n%s", value, compose)
		}
	}

	entrypointPath := filepath.Join(outputDir, docker.EntrypointOutputPath)

	entrypointContent, err := os.ReadFile(entrypointPath)
	if err != nil {
		t.Fatalf("expected generated Docker entrypoint: %v", err)
	}

	entrypoint := string(entrypointContent)

	if !strings.Contains(entrypoint, "#!/bin/sh") {
		t.Errorf("generated entrypoint does not contain shell shebang:\n%s", entrypoint)
	}

	if !strings.Contains(entrypoint, "freeradius -f") {
		t.Errorf("generated entrypoint does not contain FreeRADIUS startup command:\n%s", entrypoint)
	}
}

func TestRunMaintenanceAccountingHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := Run([]string{"maintenance", "accounting", "--help"}, &stdout, &stderr, testTemplateLoader(t), testSchemaLoader(t), nil); exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout.String(), "maintenance accounting <config.yaml> <tenant>") {
		t.Fatalf("stdout = %q, want accounting maintenance usage", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunMaintenanceAccountingUnknownTenant(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("global_objects: {}\ntenants: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := Run([]string{"maintenance", "accounting", path, "missing"}, &stdout, &stderr, testTemplateLoader(t), testSchemaLoader(t), nil); exitCode != 1 {
		t.Fatalf("Run() exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), `tenant "missing" does not exist`) {
		t.Fatalf("stderr = %q, want missing tenant error", stderr.String())
	}
}

func TestRunMaintenanceAccountingNoEnabledPolicies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	configuration := `global_objects:
  credential_profiles:
    default:
      shared_secret: secret
  authentication_profiles:
    default: {}
  accounting_profiles:
    default: {}
  monitoring_profiles:
    default: {}
  deployment_profiles:
    default:
      template: default
  nas_devices:
    router:
      ip_address: 192.0.2.1
      vendor: mikrotik
  trusted_radius_clients: {}
tenants:
  customer-a:
    authentication_profile: default
    deployment_profile: default
    database:
      engine: mysql
      host: db
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
      router:
        nas_device: router
        credential_profile: default
        accounting_profile: default
        monitoring_profile: default
    trusted_radius_client_assignments: {}
`
	if err := os.WriteFile(path, []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := Run([]string{"maintenance", "accounting", path, "customer-a"}, &stdout, &stderr, testTemplateLoader(t), testSchemaLoader(t), nil); exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no stale-session maintenance policies are enabled") {
		t.Fatalf("stdout = %q, want no enabled policies message", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
