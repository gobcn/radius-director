package runtime

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobcn/radius-director/resources"
	"github.com/moby/moby/client"
)

const (
	assetsDirectory       = "assets"
	templatesDirectory    = "templates"
	schemasDirectory      = "schemas"
	configDirectory       = "config"
	generatedDirectory    = "generated"
	envFileName           = ".env"
	composeFileName       = "compose.yaml"
	exampleConfigFileName = "example.yaml"
)

type Runtime struct {
	ID          string
	NetworkName string
	UID         int
	GID         int
}

var currentRuntimeIdentity = func() (int, int, error) {
	return currentRuntimeIdentityImpl()
}

func New(networkName string) (Runtime, error) {
	trimmedNetworkName := strings.TrimSpace(networkName)

	if trimmedNetworkName == "" {
		return Runtime{}, fmt.Errorf("runtime network name cannot be empty")
	}

	if trimmedNetworkName != networkName {
		return Runtime{}, fmt.Errorf("runtime network name cannot have leading or trailing whitespace")
	}

	id, err := newID()
	if err != nil {
		return Runtime{}, fmt.Errorf("generate runtime ID: %w", err)
	}

	return Runtime{
		ID:          id,
		NetworkName: networkName,
	}, nil
}

func newID() (string, error) {
	var b [16]byte

	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	// Format as a UUID-style identifier.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	), nil
}

func InitializeFilesystem(root string, runtime Runtime) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve runtime directory: %w", err)
	}

	if err := ensureRuntimeRoot(root); err != nil {
		return err
	}

	directories := []string{
		filepath.Join(root, assetsDirectory, templatesDirectory),
		filepath.Join(root, assetsDirectory, schemasDirectory),
		filepath.Join(root, configDirectory),
		filepath.Join(root, generatedDirectory),
	}

	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("create runtime directory %q: %w", directory, err)
		}
	}

	if err := writeEnvFile(root, runtime); err != nil {
		return err
	}

	if err := writeExampleConfig(root); err != nil {
		return err
	}

	if err := writeComposeFile(root); err != nil {
		return err
	}

	return nil
}

func Init(
	ctx context.Context,
	root string,
	networkName string,
	networkClient NetworkClient,
) error {
	runtime, err := New(networkName)
	if err != nil {
		return err
	}

	runtime.UID, runtime.GID, err = currentRuntimeIdentity()
	if err != nil {
		return fmt.Errorf("determine runtime identity: %w", err)
	}

	return initialize(ctx, root, runtime, networkClient)
}

func initialize(
	ctx context.Context,
	root string,
	runtime Runtime,
	networkClient NetworkClient,
) error {
	if err := ensureRuntimeRoot(root); err != nil {
		return err
	}

	networkCreated, err := EnsureDockerNetwork(ctx, networkClient, runtime)
	if err != nil {
		return err
	}

	if err := InitializeFilesystem(root, runtime); err != nil {
		var cleanupErrors []error

		if networkCreated {
			if _, removeErr := networkClient.NetworkRemove(
				ctx,
				runtime.NetworkName,
				client.NetworkRemoveOptions{},
			); removeErr != nil {
				cleanupErrors = append(
					cleanupErrors,
					fmt.Errorf("remove newly-created Docker network: %w", removeErr),
				)
			}
		}

		if cleanupErr := cleanupRuntimeContents(root); cleanupErr != nil {
			cleanupErrors = append(cleanupErrors, cleanupErr)
		}

		if len(cleanupErrors) > 0 {
			return fmt.Errorf(
				"initialize runtime filesystem: %v; cleanup failed: %v",
				err,
				errors.Join(cleanupErrors...),
			)
		}

		return err
	}

	return nil
}

func cleanupRuntimeContents(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve runtime directory: %w", err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("inspect runtime directory %q: %w", root, err)
	}

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())

		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove runtime path %q: %w", path, err)
		}
	}

	return nil
}

func ensureRuntimeRoot(root string) error {
	info, err := os.Stat(root)

	switch {
	case os.IsNotExist(err):
		if err := os.MkdirAll(root, 0o755); err != nil {
			return fmt.Errorf("create runtime directory %q: %w", root, err)
		}
		return nil

	case err != nil:
		return fmt.Errorf("inspect runtime directory %q: %w", root, err)

	case !info.IsDir():
		return fmt.Errorf("runtime path %q is not a directory", root)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("inspect runtime directory %q: %w", root, err)
	}

	if len(entries) != 0 {
		return fmt.Errorf("runtime directory %q is not empty", root)
	}

	return nil
}

func writeEnvFile(
	root string,
	runtime Runtime,
) error {
	path := filepath.Join(root, envFileName)

	content := fmt.Sprintf(
		"RADIUS_DIRECTOR_RUNTIME_NETWORK=%s\n"+
			"RADIUS_DIRECTOR_UID=%d\n"+
			"RADIUS_DIRECTOR_GID=%d\n",
		runtime.NetworkName,
		runtime.UID,
		runtime.GID,
	)

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write runtime environment file: %w", err)
	}

	return nil
}

func writeExampleConfig(root string) error {
	path := filepath.Join(root, configDirectory, exampleConfigFileName)

	if err := os.WriteFile(path, resources.ExampleConfig, 0o600); err != nil {
		return fmt.Errorf("write example configuration: %w", err)
	}

	return nil
}

func writeComposeFile(root string) error {
	path := filepath.Join(root, composeFileName)

	content := `services:
  radius-director:
    image: gobcn/radius-director:latest
    user: "${RADIUS_DIRECTOR_UID}:${RADIUS_DIRECTOR_GID}"
    environment:
      RADIUS_DIRECTOR_ASSETS: /assets
      RADIUS_DIRECTOR_RUNTIME_NETWORK: ${RADIUS_DIRECTOR_RUNTIME_NETWORK}
    volumes:
      - ./assets/templates:/assets/templates
      - ./assets/schemas:/assets/schemas
      - ./config:/config
      - ./generated:/generated
    networks:
      - radius-director

networks:
  radius-director:
    external: true
    name: ${RADIUS_DIRECTOR_RUNTIME_NETWORK}
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write runtime Compose file: %w", err)
	}

	return nil
}
