package services

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
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
		"get": func(m map[string]any, key string) any {
			return m[key]
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

	// Generate pages
	// Filter out the blog-post template page
	var filteredPages []models.Page
	for _, page := range project.Pages {
		if page.ID != "blog-post" {
			filteredPages = append(filteredPages, page)
		}
	}
	var totalToRender int = len(filteredPages) + len(project.BlogPosts)

	totalPages := len(filteredPages)
	for i, page := range filteredPages {
		progress := (i * 100) / totalPages
		updateProgress(progress, fmt.Sprintf("Generating static page: %d of %d ...", i+1, totalToRender))

		// Create a buffer for the page
		var pageBuf bytes.Buffer

		// Execute the template with the page data
		err := s.templates.ExecuteTemplate(&pageBuf, page.Layout, struct {
			models.Project
			Page               models.Page
			GetComponentConfig func(string) models.ComponentConfig
			BlogPost           *models.BlogPost
		}{
			Project: project,
			Page:    page,
			GetComponentConfig: func(componentType string) models.ComponentConfig {
				switch componentType {
				case "header":
					return page.Config.Header
				case "footer":
					return page.Config.Footer
				case "main":
					return page.Config.Main
				default:
					return models.ComponentConfig{}
				}
			},
			BlogPost: nil,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate page %s: %v", page.Name, err)
		}

		// Generate filename based on path
		filename := strings.TrimPrefix(page.Path, "/")
		if filename == "" {
			filename = "index.html"
		} else {
			filename = filename + ".html"
		}
		htmlFiles[filename] = pageBuf.Bytes()
	}

	// Generate blog post pages
	totalPosts := len(project.BlogPosts)
	for i, post := range project.BlogPosts {
		progress := 50 + (i * 50 / totalPosts)
		updateProgress(progress, fmt.Sprintf("Generating static page: %d of %d ...", totalPages+i+1, totalToRender))

		// Find the blog post layout page
		var blogLayoutPage *models.Page
		for _, page := range project.Pages {
			if page.ID == "blog-post" {
				blogLayoutPage = &page
				break
			}
		}
		if blogLayoutPage == nil {
			return nil, fmt.Errorf("blog post layout page not found")
		}

		// Create a buffer for the blog post page
		var postBuf bytes.Buffer

		// Execute the template with the blog post data
		err := s.templates.ExecuteTemplate(&postBuf, blogLayoutPage.Layout, struct {
			models.Project
			Page               models.Page
			GetComponentConfig func(string) models.ComponentConfig
			BlogPost           *models.BlogPost
		}{
			Project: project,
			Page:    *blogLayoutPage,
			GetComponentConfig: func(componentType string) models.ComponentConfig {
				switch componentType {
				case "header":
					return blogLayoutPage.Config.Header
				case "footer":
					return blogLayoutPage.Config.Footer
				case "main":
					return blogLayoutPage.Config.Main
				default:
					return models.ComponentConfig{}
				}
			},
			BlogPost: &post,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate blog post %s: %v", post.Title, err)
		}

		// Generate a URL-friendly filename from the title
		filename := strings.ToLower(strings.ReplaceAll(post.Title, " ", "-")) + ".html"
		htmlFiles[filename] = postBuf.Bytes()
	}

	return htmlFiles, nil
}
