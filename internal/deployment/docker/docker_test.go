package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gobcn/radius-director/internal/generator"
	"github.com/gobcn/radius-director/internal/output"
	"github.com/gobcn/radius-director/internal/schemas"
	"github.com/gobcn/radius-director/internal/templates"
)

func TestGenerateDeployment(t *testing.T) {
	directory, err := filepath.Abs(filepath.Join("..", "..", "..", "templates"))
	if err != nil {
		t.Fatalf("resolve template directory: %v", err)
	}

	loader := templates.NewLoader(os.DirFS(directory))

	schemaDirectory, err := filepath.Abs(filepath.Join("..", "..", "..", "schemas"))
	if err != nil {
		t.Fatalf("resolve schema directory: %v", err)
	}

	schemaLoader := schemas.NewLoader(os.DirFS(schemaDirectory))

	configuration := generator.Configuration{
		Tenants: []generator.Tenant{
			{
				Identifier:         "customer-a",
				DatabaseDeployment: "container",
				SQL: generator.SQL{
					Engine:   "mysql",
					Database: "customer_a",
					Username: "radius",
					Password: "database-password",
				},
				RADIUSServer: generator.RADIUSServer{
					Version:            "3.2.10",
					AuthenticationPort: 11812,
					AccountingPort:     11813,
					COAPort:            13799,
				},
			},
			{
				Identifier:         "customer-b",
				DatabaseDeployment: "external",
				RADIUSServer: generator.RADIUSServer{
					Version: "3.3.0",
				},
			},
		},
	}

	deployment, err := GenerateDeployment(loader, schemaLoader, configuration)
	if err != nil {
		t.Fatalf("GenerateDeployment() error = %v", err)
	}

	if len(deployment.Files) != 3 {
		t.Fatalf("GenerateDeployment() returned %d files, want 3", len(deployment.Files))
	}

	files := make(map[string]output.File)

	for _, file := range deployment.Files {
		files[file.Path] = file
	}

	compose, ok := files[ComposeOutputPath]
	if !ok {
		t.Fatalf("generated deployment does not contain %q", ComposeOutputPath)
	}

	expectedComposeContent := []string{
		"radius-customer-a:",
		"image: freeradius/freeradius-server:3.2.10",
		"11812:1812/udp",
		"11813:1813/udp",
		"13799:3799/udp",
		"radius-customer-b:",
		"image: freeradius/freeradius-server:3.3.0",

		"database-customer-a:",
		"image: mysql:8.4",
		"MYSQL_RANDOM_ROOT_PASSWORD: \"yes\"",
		"MYSQL_DATABASE: customer_a",
		"MYSQL_USER: radius",
		"MYSQL_PASSWORD: database-password",
		"database-customer-a-data:/var/lib/mysql",
		"./customer-a/database/schema.sql:/docker-entrypoint-initdb.d/01-radius-schema.sql:ro",

		"condition: service_healthy",
	}

	for _, value := range expectedComposeContent {
		if !strings.Contains(compose.Content, value) {
			t.Errorf("generated Docker Compose file does not contain %q:\n%s", value, compose.Content)
		}
	}

	if strings.Contains(compose.Content, "database-customer-b:") {
		t.Error("generated Docker Compose file unexpectedly contains database-customer-b")
	}

	if strings.Contains(compose.Content, "database-customer-b-data:") {
		t.Error("generated Docker Compose file unexpectedly contains database-customer-b-data")
	}

	entrypoint, ok := files[EntrypointOutputPath]
	if !ok {
		t.Fatalf("generated deployment does not contain %q", EntrypointOutputPath)
	}

	if entrypoint.Permissions != 0o755 {
		t.Errorf("entrypoint permissions = %o, want 755", entrypoint.Permissions)
	}

	if !strings.Contains(entrypoint.Content, "#!/bin/sh") {
		t.Errorf("generated entrypoint does not contain shell shebang:\n%s", entrypoint.Content)
	}

	if !strings.Contains(entrypoint.Content, "freeradius -f") {
		t.Errorf("generated entrypoint does not contain FreeRADIUS startup command:\n%s", entrypoint.Content)
	}

	schemaPath := filepath.Join("customer-a", "database", "schema.sql")

	schema, ok := files[schemaPath]
	if !ok {
		t.Fatalf("generated deployment does not contain %q", schemaPath)
	}

	if !strings.Contains(schema.Content, "CREATE TABLE IF NOT EXISTS radacct") {
		t.Errorf(
			"generated MySQL schema does not contain radacct table definition:\n%s",
			schema.Content,
		)
	}

	externalSchemaPath := filepath.Join("customer-b", "database", "schema.sql")

	if _, ok := files[externalSchemaPath]; ok {
		t.Errorf("generated deployment unexpectedly contains %q", externalSchemaPath)
	}
}

func TestGenerateDeploymentWithRuntimeNetwork(t *testing.T) {
	t.Setenv("RADIUS_DIRECTOR_RUNTIME_NETWORK", "radius-director-test")

	directory, err := filepath.Abs(filepath.Join("..", "..", "..", "templates"))
	if err != nil {
		t.Fatalf("resolve template directory: %v", err)
	}

	loader := templates.NewLoader(os.DirFS(directory))

	schemaDirectory, err := filepath.Abs(filepath.Join("..", "..", "..", "schemas"))
	if err != nil {
		t.Fatalf("resolve schema directory: %v", err)
	}

	schemaLoader := schemas.NewLoader(os.DirFS(schemaDirectory))

	configuration := generator.Configuration{
		Tenants: []generator.Tenant{
			{
				Identifier:         "customer-a",
				DatabaseDeployment: "container",
				SQL: generator.SQL{
					Engine:   "mysql",
					Database: "customer_a",
					Username: "radius",
					Password: "database-password",
				},
				RADIUSServer: generator.RADIUSServer{
					Version: "3.2.10",
				},
			},
		},
	}

	deployment, err := GenerateDeployment(loader, schemaLoader, configuration)
	if err != nil {
		t.Fatalf("GenerateDeployment() error = %v", err)
	}

	files := make(map[string]output.File)

	for _, file := range deployment.Files {
		files[file.Path] = file
	}

	compose, ok := files[ComposeOutputPath]
	if !ok {
		t.Fatalf("generated deployment does not contain %q", ComposeOutputPath)
	}

	expectedComposeContent := []string{
		"radius-customer-a:",
		"database-customer-a:",

		"    networks:",
		"      - radius-director",

		"networks:",
		"  radius-director:",
		"    external: true",
		"    name: radius-director-test",
	}

	for _, value := range expectedComposeContent {
		if !strings.Contains(compose.Content, value) {
			t.Errorf(
				"generated Docker Compose file does not contain %q:\n%s",
				value,
				compose.Content,
			)
		}
	}

	if strings.Contains(compose.Content, "${RADIUS_DIRECTOR_RUNTIME_NETWORK}") {
		t.Error(
			"generated Docker Compose file unexpectedly contains unresolved RADIUS_DIRECTOR_RUNTIME_NETWORK reference",
		)
	}
}

func TestGenerateDeploymentWithProxySQL(t *testing.T) {
	t.Setenv("RADIUS_DIRECTOR_RUNTIME_NETWORK", "radius-director-test")
	directory, err := filepath.Abs(filepath.Join("..", "..", "..", "templates"))
	if err != nil {
		t.Fatalf("resolve template directory: %v", err)
	}

	loader := templates.NewLoader(os.DirFS(directory))

	schemaDirectory, err := filepath.Abs(filepath.Join("..", "..", "..", "schemas"))
	if err != nil {
		t.Fatalf("resolve schema directory: %v", err)
	}

	schemaLoader := schemas.NewLoader(os.DirFS(schemaDirectory))

	configuration := generator.Configuration{
		Tenants: []generator.Tenant{
			{
				Identifier:         "customer-a",
				DatabaseDeployment: "proxysql",
				SQL: generator.SQL{
					Engine:   "mysql",
					Database: "radius_a",
					Username: "radius_a",
					Password: "password-a",
				},
				ProxySQL: &generator.ProxySQL{
					BackendHost: "database-a.example.com",
					BackendPort: 3306,
				},
				RADIUSServer: generator.RADIUSServer{
					Version: "3.2.10",
				},
			},
			{
				Identifier:         "customer-b",
				DatabaseDeployment: "proxysql",
				SQL: generator.SQL{
					Engine:   "mysql",
					Database: "radius_b",
					Username: "radius_b",
					Password: "password-b",
				},
				ProxySQL: &generator.ProxySQL{
					BackendHost: "database-b.example.com",
					BackendPort: 3307,
				},
				RADIUSServer: generator.RADIUSServer{
					Version: "3.3.0",
				},
			},
		},
	}

	deployment, err := GenerateDeployment(loader, schemaLoader, configuration)
	if err != nil {
		t.Fatalf("GenerateDeployment() error = %v", err)
	}

	if len(deployment.Files) != 3 {
		t.Fatalf(
			"GenerateDeployment() returned %d files, want 3",
			len(deployment.Files),
		)
	}

	files := make(map[string]output.File)

	for _, file := range deployment.Files {
		files[file.Path] = file
	}

	compose, ok := files[ComposeOutputPath]
	if !ok {
		t.Fatalf("generated deployment does not contain %q", ComposeOutputPath)
	}

	composeContent := strings.ReplaceAll(compose.Content, "\r\n", "\n")

	expectedComposeContent := []string{
		"proxysql:",
		"radius-customer-a:",
		"image: freeradius/freeradius-server:3.2.10",
		"radius-customer-b:",
		"image: freeradius/freeradius-server:3.3.0",

		"    networks:",
		"      - radius-director",

		"networks:",
		"  radius-director:",
		"    external: true",
		"    name: radius-director-test",
	}

	for _, value := range expectedComposeContent {
		if !strings.Contains(composeContent, value) {
			t.Errorf(
				"generated Docker Compose file does not contain %q:\n%s",
				value,
				compose.Content,
			)
		}
	}

	if strings.Contains(composeContent, "proxysql-customer-a:") {
		t.Error(
			"generated Docker Compose file unexpectedly contains per-tenant ProxySQL service",
		)
	}

	if strings.Contains(composeContent, "proxysql-customer-b:") {
		t.Error(
			"generated Docker Compose file unexpectedly contains per-tenant ProxySQL service",
		)
	}

	if strings.Contains(composeContent, "\nvolumes:\n") {
		t.Errorf(
			"generated Docker Compose file unexpectedly contains a top-level volumes section",
		)
	}

	proxySQL, ok := files[ProxySQLOutputPath]
	if !ok {
		t.Fatalf(
			"generated deployment does not contain %q",
			ProxySQLOutputPath,
		)
	}

	expectedProxySQLContent := []string{
		"database-a.example.com",
		"3306",
		"hostgroup=10",
		"radius_a",
		"password-a",
		"database-b.example.com",
		"3307",
		"hostgroup=11",
		"radius_b",
		"password-b",
	}

	for _, value := range expectedProxySQLContent {
		if !strings.Contains(proxySQL.Content, value) {
			t.Errorf(
				"generated ProxySQL configuration does not contain %q:\n%s",
				value,
				proxySQL.Content,
			)
		}
	}
}

func TestGenerateDeploymentWithoutRuntimeNetwork(t *testing.T) {
	t.Setenv("RADIUS_DIRECTOR_RUNTIME_NETWORK", "")

	directory, err := filepath.Abs(filepath.Join("..", "..", "..", "templates"))
	if err != nil {
		t.Fatalf("resolve template directory: %v", err)
	}

	loader := templates.NewLoader(os.DirFS(directory))

	schemaDirectory, err := filepath.Abs(filepath.Join("..", "..", "..", "schemas"))
	if err != nil {
		t.Fatalf("resolve schema directory: %v", err)
	}

	schemaLoader := schemas.NewLoader(os.DirFS(schemaDirectory))

	configuration := generator.Configuration{
		Tenants: []generator.Tenant{
			{
				Identifier:         "customer-a",
				DatabaseDeployment: "container",
				SQL: generator.SQL{
					Engine:   "mysql",
					Database: "customer_a",
					Username: "radius",
					Password: "database-password",
				},
				RADIUSServer: generator.RADIUSServer{
					Version: "3.2.10",
				},
			},
		},
	}

	deployment, err := GenerateDeployment(loader, schemaLoader, configuration)
	if err != nil {
		t.Fatalf("GenerateDeployment() error = %v", err)
	}

	files := make(map[string]output.File)

	for _, file := range deployment.Files {
		files[file.Path] = file
	}

	compose, ok := files[ComposeOutputPath]
	if !ok {
		t.Fatalf("generated deployment does not contain %q", ComposeOutputPath)
	}

	if strings.Contains(compose.Content, "RADIUS_DIRECTOR_RUNTIME_NETWORK") {
		t.Error(
			"generated Docker Compose unexpectedly references RADIUS_DIRECTOR_RUNTIME_NETWORK",
		)
	}

	if strings.Contains(compose.Content, "radius-director:") {
		t.Error(
			"generated Docker Compose unexpectedly contains a radius-director network",
		)
	}

	if strings.Contains(compose.Content, "      - radius-director") {
		t.Error(
			"generated Docker Compose unexpectedly attaches services to the radius-director network",
		)
	}
}

func TestNewComposeDataRuntimeNetwork(t *testing.T) {
	t.Setenv("RADIUS_DIRECTOR_RUNTIME_NETWORK", "radius-director-test")

	data := NewComposeData(generator.Configuration{})

	if data.RuntimeNetworkName != "radius-director-test" {
		t.Errorf(
			"RuntimeNetworkName = %q, want %q",
			data.RuntimeNetworkName,
			"radius-director-test",
		)
	}
}

func TestNewComposeDataWithoutRuntimeNetwork(t *testing.T) {
	t.Setenv("RADIUS_DIRECTOR_RUNTIME_NETWORK", "")

	data := NewComposeData(generator.Configuration{})

	if data.RuntimeNetworkName != "" {
		t.Errorf(
			"RuntimeNetworkName = %q, want empty string",
			data.RuntimeNetworkName,
		)
	}
}

func TestNewComposeDataProxySQL(t *testing.T) {
	configuration := generator.Configuration{
		Tenants: []generator.Tenant{
			{
				Identifier:         "customer-a",
				DatabaseDeployment: "container",
			},
			{
				Identifier:         "customer-b",
				DatabaseDeployment: "proxysql",
				ProxySQL: &generator.ProxySQL{
					BackendHost: "database-b.example.com",
					BackendPort: 3306,
				},
				SQL: generator.SQL{
					Database: "radius_b",
					Username: "radius_b",
					Password: "secret-b",
				},
			},
			{
				Identifier:         "customer-c",
				DatabaseDeployment: "proxysql",
				ProxySQL: &generator.ProxySQL{
					BackendHost: "database-c.example.com",
					BackendPort: 3307,
				},
				SQL: generator.SQL{
					Database: "radius_c",
					Username: "radius_c",
					Password: "secret-c",
				},
			},
		},
	}

	data := NewComposeData(configuration)

	if data.ProxySQL == nil {
		t.Fatal("ProxySQL is nil, want non-nil")
	}

	if data.ProxySQL.ServiceName != "proxysql" {
		t.Errorf(
			"ProxySQL.ServiceName = %q, want %q",
			data.ProxySQL.ServiceName,
			"proxysql",
		)
	}

	if len(data.ProxySQL.Backends) != 2 {
		t.Fatalf(
			"len(ProxySQL.Backends) = %d, want 2",
			len(data.ProxySQL.Backends),
		)
	}

	first := data.ProxySQL.Backends[0]

	if first.TenantName != "customer-b" {
		t.Errorf("first TenantName = %q, want %q", first.TenantName, "customer-b")
	}

	if first.Host != "database-b.example.com" {
		t.Errorf("first Host = %q, want %q", first.Host, "database-b.example.com")
	}

	if first.Port != 3306 {
		t.Errorf("first Port = %d, want 3306", first.Port)
	}

	if first.Database != "radius_b" {
		t.Errorf("first Database = %q, want %q", first.Database, "radius_b")
	}

	if first.Username != "radius_b" {
		t.Errorf("first Username = %q, want %q", first.Username, "radius_b")
	}

	if first.Password != "secret-b" {
		t.Errorf("first Password = %q, want %q", first.Password, "secret-b")
	}

	if first.Hostgroup != 10 {
		t.Errorf("first Hostgroup = %d, want 10", first.Hostgroup)
	}

	second := data.ProxySQL.Backends[1]

	if second.TenantName != "customer-c" {
		t.Errorf("second TenantName = %q, want %q", second.TenantName, "customer-c")
	}

	if second.Host != "database-c.example.com" {
		t.Errorf("second Host = %q, want %q", second.Host, "database-c.example.com")
	}

	if second.Port != 3307 {
		t.Errorf("second Port = %d, want 3307", second.Port)
	}

	if second.Hostgroup != 11 {
		t.Errorf("second Hostgroup = %d, want 11", second.Hostgroup)
	}
}

func TestCompactComposeWhitespace(t *testing.T) {
	input := "services:\n\n\n  proxysql:\n    image: proxysql/proxysql:latest\n\n\n\n  radius-customer-a:\n    image: freeradius/freeradius-server:3.2.10\n\nnetworks:\n"

	expected := "services:\n\n  proxysql:\n    image: proxysql/proxysql:latest\n\n  radius-customer-a:\n    image: freeradius/freeradius-server:3.2.10\n\nnetworks:\n"

	actual := compactComposeWhitespace(input)

	if actual != expected {
		t.Errorf("compactComposeWhitespace() = %q, want %q", actual, expected)
	}
}
