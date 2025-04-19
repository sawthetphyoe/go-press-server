package services

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

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
		"formatDate": func(date string) string {
			t, err := time.Parse("2006-01-02", date)
			if err != nil {
				return date
			}
			return t.Format("January 2, 2006")
		},
		"now": func() int {
			return time.Now().Year()
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"get": func(m map[string]any, key string) any {
			return m[key]
		},
		"isLinkActive": func(href string, page models.Page) bool {
			var isActive bool = false
			if page.Slug == "/" {
				isActive = href == "./index.html"
			} else {
				isActive = href == "."+page.Slug+".html"
			}
			return isActive
		},
		"getYear": func() int {
			return time.Now().Year()
		},
	})

	// Parse all template files
	patterns := []string{
		"internal/templates/atoms/*.tmpl",
		"internal/templates/molecules/*.tmpl",
		"internal/templates/organisms/*.tmpl",
		"internal/templates/layouts/*.tmpl",
		"internal/templates/*.tmpl",
	}

	for _, pattern := range patterns {
		_, err := tmpl.ParseGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to parse templates: %v", err)
		}
	}

	return &TemplateService{
		templates: tmpl,
	}, nil
}

// GenerateHTML generates HTML from the project data
func (s *TemplateService) GenerateHTML(project models.Project, updateProgress func(int, string)) (map[string][]byte, error) {
	// Create a map to store all HTML files
	htmlFiles := make(map[string][]byte)

	// Generate pages
	var totalToRender int = len(project.Pages)

	totalPages := len(project.Pages)
	for i, page := range project.Pages {
		progress := (i * 100) / totalPages
		updateProgress(progress, fmt.Sprintf("Generating static page: %d of %d ...", i+1, totalToRender))

		// Create a buffer for the page
		var pageBuf bytes.Buffer

		// Execute the template with the page data
		err := s.templates.ExecuteTemplate(&pageBuf, "layouts/default", struct {
			models.Project
			Page models.Page
		}{
			Project: project,
			Page:    page,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate page %s: %v", page.Title, err)
		}

		// Generate filename based on slug
		filename := strings.TrimPrefix(page.Slug, "/")
		if filename == "" {
			filename = "index.html"
		} else {
			filename = filename + ".html"
		}
		htmlFiles[filename] = pageBuf.Bytes()
	}

	return htmlFiles, nil
}
