package services

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"sawthet.go-press-server.net/internal/models"
)

// TemplateService handles template generation
type TemplateService struct {
	templates *template.Template
}

// NewTemplateService creates a new template service
func NewTemplateService() (*TemplateService, error) {
	// Create a new template with our custom functions
	tmpl := template.New("base").Funcs(template.FuncMap{
		"now": func() time.Time { return time.Now() },
	})

	// Define the template files
	templateFiles := []string{
		"internal/templates/base.tmpl",
		"internal/templates/header.tmpl",
		"internal/templates/blog_post.tmpl",
		"internal/templates/footer.tmpl",
	}

	// Parse all templates
	tmpl, err := tmpl.ParseFiles(templateFiles...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}

	return &TemplateService{
		templates: tmpl,
	}, nil
}

// GenerateHTML generates HTML from the project data
func (s *TemplateService) GenerateHTML(project models.Project) ([]byte, error) {
	var buffer bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buffer, "base", project); err != nil {
		return nil, fmt.Errorf("error executing template: %v", err)
	}
	return buffer.Bytes(), nil
}
