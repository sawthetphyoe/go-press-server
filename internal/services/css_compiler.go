package services

import (
	"os"
	"os/exec"
	"path/filepath"
)

// CSSCompiler handles the compilation of Tailwind CSS
type CSSCompiler struct {
	tempDir string
}

// NewCSSCompiler creates a new CSS compiler instance
func NewCSSCompiler() (*CSSCompiler, error) {
	tempDir, err := os.MkdirTemp("", "css-compiler-*")
	if err != nil {
		return nil, err
	}

	return &CSSCompiler{
		tempDir: tempDir,
	}, nil
}

// Compile generates Tailwind CSS based on the provided HTML content
func (c *CSSCompiler) Compile(htmlContent []byte) ([]byte, error) {
	// Create input HTML file
	htmlPath := filepath.Join(c.tempDir, "input.html")
	if err := os.WriteFile(htmlPath, htmlContent, 0644); err != nil {
		return nil, err
	}

	// Create input CSS file
	cssPath := filepath.Join(c.tempDir, "input.css")
	if err := os.WriteFile(cssPath, []byte("@tailwind base;\n@tailwind components;\n@tailwind utilities;"), 0644); err != nil {
		return nil, err
	}

	// Create Tailwind config
	configPath := filepath.Join(c.tempDir, "tailwind.config.js")
	configContent := `module.exports = {
  content: ["input.html"],
  theme: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return nil, err
	}

	// Create package.json
	packagePath := filepath.Join(c.tempDir, "package.json")
	packageContent := `{
  "dependencies": {
    "tailwindcss": "^3.4.1",
    "@tailwindcss/typography": "^0.5.10"
  }
}`
	if err := os.WriteFile(packagePath, []byte(packageContent), 0644); err != nil {
		return nil, err
	}

	// Install dependencies
	cmd := exec.Command("npm", "install")
	cmd.Dir = c.tempDir
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Compile CSS
	outputPath := filepath.Join(c.tempDir, "output.css")
	cmd = exec.Command("npx", "tailwindcss", "-i", "input.css", "-o", "output.css", "--minify")
	cmd.Dir = c.tempDir
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Read compiled CSS
	compiledCSS, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}

	return compiledCSS, nil
}

// Cleanup removes temporary files
func (c *CSSCompiler) Cleanup() error {
	return os.RemoveAll(c.tempDir)
}
