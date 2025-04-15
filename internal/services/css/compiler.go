package css

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"sawthet.go-press-server.net/internal/services/css/shared"
)

type CSSCompiler struct {
	tempDir string
}

func NewCSSCompiler() (*CSSCompiler, error) {
	// Initialize shared node_modules
	if err := shared.Setup(shared.Config{}); err != nil {
		return nil, fmt.Errorf("failed to setup shared node_modules: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "css-compiler-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Create symlink to shared node_modules
	if err := os.Symlink(
		shared.GetNodeModulesPath(""),
		filepath.Join(tempDir, "node_modules"),
	); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to create node_modules symlink: %v", err)
	}

	return &CSSCompiler{
		tempDir: tempDir,
	}, nil
}

func (c *CSSCompiler) Compile(htmlContent []byte) ([]byte, error) {
	// Create input.css
	inputCSS := `@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
	html {
		@apply antialiased;
	}
}`
	if err := os.WriteFile(filepath.Join(c.tempDir, "input.css"), []byte(inputCSS), 0644); err != nil {
		return nil, fmt.Errorf("failed to create input.css: %v", err)
	}

	// Create tailwind.config.js
	configContent := `module.exports = {
		content: ["./*.html"],
		theme: {
			extend: {},
		},
		plugins: [
			require('@tailwindcss/typography'),
		],
	}`
	if err := os.WriteFile(filepath.Join(c.tempDir, "tailwind.config.js"), []byte(configContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create tailwind.config.js: %v", err)
	}

	// Create input.html
	if err := os.WriteFile(filepath.Join(c.tempDir, "input.html"), htmlContent, 0644); err != nil {
		return nil, fmt.Errorf("failed to create input.html: %v", err)
	}

	// Compile CSS
	cmd := exec.Command("npx", "tailwindcss", "-i", "input.css", "-o", "output.css")
	cmd.Dir = c.tempDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to compile CSS: %v, stderr: %s", err, stderr.String())
	}

	// Read compiled CSS
	compiledCSS, err := os.ReadFile(filepath.Join(c.tempDir, "output.css"))
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled CSS: %v", err)
	}

	return compiledCSS, nil
}

func (c *CSSCompiler) Cleanup() error {
	return os.RemoveAll(c.tempDir)
}
