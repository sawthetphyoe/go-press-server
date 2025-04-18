package services

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"sawthet.go-press-server.net/internal/models"
	"sawthet.go-press-server.net/internal/services/css/shared"
)

// CSSCompiler handles the compilation of Tailwind CSS
type CSSCompiler struct {
	tempDir string
}

// NewCSSCompiler creates a new CSS compiler instance
func NewCSSCompiler() (*CSSCompiler, error) {
	// Setup shared node modules
	if err := shared.Setup(shared.Config{
		NodeDir: "internal/services/css/shared",
	}); err != nil {
		return nil, fmt.Errorf("failed to setup shared node modules: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "css-compiler-*")
	if err != nil {
		return nil, err
	}

	return &CSSCompiler{
		tempDir: tempDir,
	}, nil
}

// generateTailwindConfig creates a Tailwind config file with theme values
func (c *CSSCompiler) generateTailwindConfig(project models.Project) error {
	configContent := fmt.Sprintf(`module.exports = {
  content: ["./**/*.html"],
  theme: {
    extend: {
      colors: {
        primary: "%s",
        secondary: "%s",
        background: "%s",
        text: "%s"
      },
      fontFamily: {
        sans: ["%s", "sans-serif"]
      },
      fontSize: {
        sm: "%s",
        base: "%s",
        lg: "%s",
        xl: "%s",
        "2xl": "%s"
      },
      spacing: {
        sm: "%s",
        md: "%s",
        lg: "%s",
        xl: "%s"
      }
    }
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}`,
		project.GlobalConfig.Theme.Colors.Primary,
		project.GlobalConfig.Theme.Colors.Secondary,
		project.GlobalConfig.Theme.Colors.Background,
		project.GlobalConfig.Theme.Colors.Text,
		project.GlobalConfig.Theme.Typography.FontFamily,
		project.GlobalConfig.Theme.Typography.FontSizes.Small,
		project.GlobalConfig.Theme.Typography.FontSizes.Base,
		project.GlobalConfig.Theme.Typography.FontSizes.Large,
		project.GlobalConfig.Theme.Typography.FontSizes.XLarge,
		project.GlobalConfig.Theme.Typography.FontSizes.XXLarge,
		project.GlobalConfig.Theme.Spacing.Small,
		project.GlobalConfig.Theme.Spacing.Medium,
		project.GlobalConfig.Theme.Spacing.Large,
		project.GlobalConfig.Theme.Spacing.XLarge,
	)

	configPath := filepath.Join(c.tempDir, "tailwind.config.js")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}

	return nil
}

// Compile compiles the CSS using Tailwind CSS
func (c *CSSCompiler) Compile(htmlContent []byte, project models.Project) ([]byte, error) {
	// Create input HTML file
	htmlPath := filepath.Join(c.tempDir, "input.html")
	if err := os.WriteFile(htmlPath, htmlContent, 0644); err != nil {
		return nil, err
	}

	// Create input CSS file with Tailwind directives
	cssPath := filepath.Join(c.tempDir, "input.css")
	cssContent := `@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --color-primary: ` + project.GlobalConfig.Theme.Colors.Primary + `;
    --color-secondary: ` + project.GlobalConfig.Theme.Colors.Secondary + `;
    --color-background: ` + project.GlobalConfig.Theme.Colors.Background + `;
    --color-text: ` + project.GlobalConfig.Theme.Colors.Text + `;
  }
}

@layer components {
  .bg-primary { background-color: var(--color-primary); }
  .text-primary { color: var(--color-primary); }
  .bg-secondary { background-color: var(--color-secondary); }
  .text-secondary { color: var(--color-secondary); }
  .bg-background { background-color: var(--color-background); }
  .text-text { color: var(--color-text); }
}`

	if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
		return nil, err
	}

	// Generate Tailwind config with theme values
	if err := c.generateTailwindConfig(project); err != nil {
		return nil, err
	}

	// Copy node_modules to temp directory
	nodeModulesPath := shared.GetNodeModulesPath("")
	tempNodeModules := filepath.Join(c.tempDir, "node_modules")
	if err := os.MkdirAll(tempNodeModules, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp node_modules: %v", err)
	}

	// Copy the entire node_modules directory
	cmd := exec.Command("cp", "-r", nodeModulesPath+"/.", tempNodeModules)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to copy node_modules: %v", err)
	}

	// Compile CSS using local node_modules with minification
	outputPath := filepath.Join(c.tempDir, "output.css")
	tailwindPath := filepath.Join(tempNodeModules, "tailwindcss", "lib", "cli.js")
	cmd = exec.Command("node", tailwindPath, "-i", "input.css", "-o", "output.css", "--content", "input.html", "--minify")
	cmd.Dir = c.tempDir
	cmd.Env = append(os.Environ(), "NODE_PATH="+tempNodeModules)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to compile CSS: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
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
