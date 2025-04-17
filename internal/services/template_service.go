package services

import (
	"bytes"
	"errors"
	"html/template"
	"path/filepath"

	"sawthet.go-press-server.net/internal/models"
)

// TemplateService handles template generation
type TemplateService struct {
	templates *template.Template
}

// NewTemplateService creates a new template service
func NewTemplateService() (*TemplateService, error) {
	// Create a new template with custom functions
	tmpl := template.New("").Funcs(template.FuncMap{
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("dict requires an even number of arguments")
			}
			dict := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	})

	// Parse all template files
	pattern := filepath.Join("internal", "templates", "*.tmpl")
	templates, err := tmpl.ParseGlob(pattern)
	if err != nil {
		return nil, err
	}

	return &TemplateService{
		templates: templates,
	}, nil
}

// GenerateHTML generates HTML from the project data
func (s *TemplateService) GenerateHTML(project models.Project) ([]byte, error) {
	// Create a buffer to write the output
	var buf bytes.Buffer

	// Execute the base template with the project data
	err := s.templates.ExecuteTemplate(&buf, "base", struct {
		models.Project
		GetComponentConfig func(string) models.ComponentConfig
	}{
		Project: project,
		GetComponentConfig: func(componentType string) models.ComponentConfig {
			switch componentType {
			case "header":
				return project.Config.Header
			case "blogPost":
				return project.Config.BlogPost
			case "footer":
				return project.Config.Footer
			case "main":
				return project.Config.Main
			default:
				return models.ComponentConfig{}
			}
		},
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
