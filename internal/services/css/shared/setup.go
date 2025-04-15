package shared

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	setupOnce sync.Once
	setupErr  error
)

const (
	DefaultNodeDir = "internal/services/css/shared"
)

type Config struct {
	NodeDir string
}

func Setup(config Config) error {
	if config.NodeDir == "" {
		config.NodeDir = DefaultNodeDir
	}

	setupOnce.Do(func() {
		setupErr = setupSharedNodeModules(config.NodeDir)
	})

	return setupErr
}

func setupSharedNodeModules(nodeDir string) error {
	// Create package.json if it doesn't exist
	packageJSONPath := filepath.Join(nodeDir, "package.json")
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		if err := os.MkdirAll(nodeDir, 0755); err != nil {
			return fmt.Errorf("failed to create shared directory: %v", err)
		}

		packageContent := `{
			"dependencies": {
				"tailwindcss": "^3.4.1",
				"@tailwindcss/typography": "^0.5.10"
			}
		}`
		if err := os.WriteFile(packageJSONPath, []byte(packageContent), 0644); err != nil {
			return fmt.Errorf("failed to create package.json: %v", err)
		}

		// Install dependencies
		cmd := exec.Command("npm", "install")
		cmd.Dir = nodeDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install dependencies: %v\nOutput: %s", err, string(output))
		}
	}

	return nil
}

func GetNodeModulesPath(nodeDir string) string {
	if nodeDir == "" {
		nodeDir = DefaultNodeDir
	}
	return filepath.Join(nodeDir, "node_modules")
}
