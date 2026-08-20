package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/gobcn/radius-director/resources"
)

func TestNewRejectsEmptyNetworkName(t *testing.T) {
	tests := []string{
		"",
		" ",
		"   ",
		"\t",
		"\n",
	}

	for _, networkName := range tests {
		t.Run(networkName, func(t *testing.T) {
			_, err := New(networkName)
			if err == nil {
				t.Fatalf("New(%q) expected an error", networkName)
			}
		})
	}
}

func TestNewPreservesNetworkName(t *testing.T) {
	const networkName = "radius-director-prod"

	runtime, err := New(networkName)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if runtime.NetworkName != networkName {
		t.Fatalf("NetworkName = %q, want %q", runtime.NetworkName, networkName)
	}
}

func TestNewGeneratesRuntimeID(t *testing.T) {
	runtime, err := New("radius-director-prod")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if runtime.ID == "" {
		t.Fatal("New() generated an empty runtime ID")
	}

	if len(runtime.ID) != 36 {
		t.Fatalf("runtime ID length = %d, want 36", len(runtime.ID))
	}

	if strings.Count(runtime.ID, "-") != 4 {
		t.Fatalf("runtime ID = %q, want UUID-style formatting", runtime.ID)
	}
}

func TestNewGeneratesUniqueRuntimeIDs(t *testing.T) {
	first, err := New("radius-director-prod")
	if err != nil {
		t.Fatalf("first New() error = %v", err)
	}

	second, err := New("radius-director-prod")
	if err != nil {
		t.Fatalf("second New() error = %v", err)
	}

	if first.ID == second.ID {
		t.Fatalf("generated identical runtime IDs: %q", first.ID)
	}
}

func TestNewRejectsNetworkNameWithSurroundingWhitespace(t *testing.T) {
	tests := []string{
		" radius-director-prod",
		"radius-director-prod ",
		" radius-director-prod ",
	}

	for _, networkName := range tests {
		t.Run(networkName, func(t *testing.T) {
			_, err := New(networkName)
			if err == nil {
				t.Fatalf("New(%q) expected an error", networkName)
			}
		})
	}
}

func TestInitializeFilesystemCreatesRuntimeStructure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")

	runtime, err := New("radius-director-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := InitializeFilesystem(root, runtime); err != nil {
		t.Fatalf("InitializeFilesystem() error = %v", err)
	}

	expectedDirectories := []string{
		filepath.Join(root, "assets"),
		filepath.Join(root, "assets", "templates"),
		filepath.Join(root, "assets", "schemas"),
		filepath.Join(root, "config"),
		filepath.Join(root, "generated"),
	}

	for _, path := range expectedDirectories {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected directory %q: %v", path, err)
			continue
		}

		if !info.IsDir() {
			t.Errorf("expected %q to be a directory", path)
		}
	}

	for _, name := range []string{".env", "compose.yaml"} {
		path := filepath.Join(root, name)

		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected file %q: %v", path, err)
			continue
		}

		if !info.Mode().IsRegular() {
			t.Errorf("expected %q to be a regular file", path)
		}
	}
}

func TestInitializeFilesystemWritesEnvironmentFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")

	runtime, err := New("radius-director-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	runtime.UID = 1234
	runtime.GID = 5678

	if err := InitializeFilesystem(root, runtime); err != nil {
		t.Fatalf("InitializeFilesystem() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}

	want :=
		"RADIUS_DIRECTOR_RUNTIME_NETWORK=radius-director-test\n" +
			"RADIUS_DIRECTOR_UID=1234\n" +
			"RADIUS_DIRECTOR_GID=5678\n"

	if string(content) != want {
		t.Fatalf(".env = %q, want %q", string(content), want)
	}
}

func TestInitializeFilesystemWritesComposeFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")

	runtime, err := New("radius-director-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := InitializeFilesystem(root, runtime); err != nil {
		t.Fatalf("InitializeFilesystem() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, "compose.yaml"))
	if err != nil {
		t.Fatalf("read compose.yaml: %v", err)
	}

	compose := string(content)

	expected := []string{
		"radius-director:",
		"image: ghcr.io/gobcn/radius-director:latest",
		"user: \"${RADIUS_DIRECTOR_UID}:${RADIUS_DIRECTOR_GID}\"",
		"RADIUS_DIRECTOR_ASSETS: /assets",
		"RADIUS_DIRECTOR_RUNTIME_NETWORK: ${RADIUS_DIRECTOR_RUNTIME_NETWORK}",
		"./assets/templates:/assets/templates",
		"./assets/schemas:/assets/schemas",
		"./config:/config",
		"./generated:/generated",
		"external: true",
		"name: ${RADIUS_DIRECTOR_RUNTIME_NETWORK}",
	}

	for _, value := range expected {
		if !strings.Contains(compose, value) {
			t.Errorf("compose.yaml does not contain %q:\n%s", value, compose)
		}
	}
}

func TestInitializeFilesystemRejectsNonEmptyDirectory(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(root, "existing.txt"),
		[]byte("existing"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	runtime, err := New("radius-director-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = InitializeFilesystem(root, runtime)
	if err == nil {
		t.Fatal("InitializeFilesystem() expected an error for non-empty directory")
	}
}

func TestInitializeFilesystemAcceptsExistingEmptyDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")

	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	runtime, err := New("radius-director-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := InitializeFilesystem(root, runtime); err != nil {
		t.Fatalf("InitializeFilesystem() error = %v", err)
	}
}

func TestInitializeFilesystemCreatesMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nested", "runtime")

	runtime, err := New("radius-director-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := InitializeFilesystem(root, runtime); err != nil {
		t.Fatalf("InitializeFilesystem() error = %v", err)
	}

	if _, err := os.Stat(root); err != nil {
		t.Fatalf("runtime root was not created: %v", err)
	}
}

func TestInitCreatesRuntime(t *testing.T) {
	originalCurrentRuntimeIdentity := currentRuntimeIdentity
	currentRuntimeIdentity = func() (int, int, error) {
		return 1234, 5678, nil
	}
	t.Cleanup(func() {
		currentRuntimeIdentity = originalCurrentRuntimeIdentity
	})
	root := filepath.Join(t.TempDir(), "runtime")

	networkName := "radius-director-test"

	fake := &fakeNetworkClient{
		inspectErrors: []error{
			errdefs.ErrNotFound,
		},
	}

	if err := Init(context.Background(), root, networkName, fake); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if fake.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", fake.createCalls)
	}

	if fake.removeCalls != 0 {
		t.Fatalf("remove calls = %d, want 0", fake.removeCalls)
	}

	if _, err := os.Stat(filepath.Join(root, ".env")); err != nil {
		t.Fatalf("expected .env: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "config", "example.yaml")); err != nil {
		t.Fatalf("expected config/example.yaml: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "compose.yaml")); err != nil {
		t.Fatalf("expected compose.yaml: %v", err)
	}
}

func TestInitRejectsNonEmptyRuntimeDirectoryBeforeCreatingNetwork(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(root, "existing.txt"),
		[]byte("existing"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	fake := &fakeNetworkClient{}

	err := Init(
		context.Background(),
		root,
		"radius-director-test",
		fake,
	)
	if err == nil {
		t.Fatal("Init() expected an error")
	}

	if fake.inspectCalls != 0 {
		t.Fatalf("inspect calls = %d, want 0", fake.inspectCalls)
	}

	if fake.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", fake.createCalls)
	}

	if fake.removeCalls != 0 {
		t.Fatalf("remove calls = %d, want 0", fake.removeCalls)
	}

	if _, err := os.Stat(filepath.Join(root, "existing.txt")); err != nil {
		t.Fatalf("existing file was unexpectedly removed: %v", err)
	}
}

func TestInitializeFilesystemWritesExampleConfig(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")

	runtime, err := New("radius-director-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := InitializeFilesystem(root, runtime); err != nil {
		t.Fatalf("InitializeFilesystem() error = %v", err)
	}

	path := filepath.Join(root, "config", "example.yaml")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example.yaml: %v", err)
	}

	if string(content) != string(resources.ExampleConfig) {
		t.Fatal("installed example.yaml does not match embedded example configuration")
	}
}
