package services

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

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
		"urlize": func(s string) string {
			return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
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
func (s *TemplateService) GenerateHTML(project models.Project, updateProgress func(int, string)) (map[string][]byte, error) {
	// Create a map to store all HTML files
	htmlFiles := make(map[string][]byte)

	// Generate index.html
	updateProgress(25, "Generating index.html...")
	var indexBuf bytes.Buffer
	err := s.templates.ExecuteTemplate(&indexBuf, "base", struct {
		models.Project
		GetComponentConfig func(string) models.ComponentConfig
		BlogPost           *models.Component
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
		BlogPost: nil,
	})
	if err != nil {
		return nil, err
	}
	htmlFiles["index.html"] = indexBuf.Bytes()

	// Generate individual blog post pages
	totalPosts := 0
	for _, component := range project.Components {
		if component.Type == "blogPost" {
			totalPosts++
		}
	}

	currentPost := 0
	for _, component := range project.Components {
		if component.Type == "blogPost" {
			currentPost++
			progress := 25 + (currentPost * 25 / totalPosts)
			updateProgress(progress, fmt.Sprintf("Generating blog page %d of %d...", currentPost, totalPosts))

			// Create a buffer for the blog post page
			var blogBuf bytes.Buffer

			// Execute the base template with the blog post data
			err := s.templates.ExecuteTemplate(&blogBuf, "base", struct {
				models.Project
				GetComponentConfig func(string) models.ComponentConfig
				BlogPost           models.Component
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
				BlogPost: component,
			})
			if err != nil {
				return nil, err
			}

			// Generate a URL-friendly filename from the title
			title := component.Content["title"].(string)
			filename := strings.ToLower(strings.ReplaceAll(title, " ", "-")) + ".html"
			htmlFiles[filename] = blogBuf.Bytes()
		}
	}

	return htmlFiles, nil
}
