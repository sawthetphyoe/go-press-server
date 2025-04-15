package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/julienschmidt/httprouter"
	"sawthet.go-press-server.net/internal/models"
	"sawthet.go-press-server.net/internal/services"
	"sawthet.go-press-server.net/internal/services/job"
)

// home is the handler for the home page
func (app *application) home(w http.ResponseWriter, r *http.Request) {

	w.Write([]byte("Hello World!"))
}

func (app *application) buildProject(w http.ResponseWriter, r *http.Request) {
	// For now, we'll use the sample project data
	projectData, err := os.ReadFile("data/sample_project.json")
	if err != nil {
		app.serverError(w, err)
		return
	}

	var project models.Project
	if err := json.Unmarshal(projectData, &project); err != nil {
		app.serverError(w, err)
		return
	}

	// Debug: Print project data
	fmt.Printf("Project Name: %s\n", project.Name)
	fmt.Printf("Number of Components: %d\n", len(project.Components))
	for i, comp := range project.Components {
		fmt.Printf("Component %d: %+v\n", i, comp)
	}

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
	tmpl, err = tmpl.ParseFiles(templateFiles...)
	if err != nil {
		app.serverError(w, fmt.Errorf("error parsing templates: %v", err))
		return
	}

	// Create the output directory
	outputDir := filepath.Join("static", "sites", "sample-project")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		app.serverError(w, err)
		return
	}

	// Create the output file
	outputFile, err := os.Create(filepath.Join(outputDir, "index.html"))
	if err != nil {
		app.serverError(w, err)
		return
	}
	defer outputFile.Close()

	// Execute the template
	if err := tmpl.ExecuteTemplate(outputFile, "base", project); err != nil {
		app.serverError(w, fmt.Errorf("error executing template: %v", err))
		return
	}

	// Create the CSS directory in the output directory
	cssDir := filepath.Join(outputDir, "css")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		app.serverError(w, err)
		return
	}

	// Copy the compiled Tailwind CSS
	sourceCSS := "static/css/tailwind.css"
	destCSS := filepath.Join(cssDir, "tailwind.css")

	// Read the source CSS file
	source, err := os.ReadFile(sourceCSS)
	if err != nil {
		app.serverError(w, fmt.Errorf("error reading source CSS: %v", err))
		return
	}

	// Write the CSS file
	if err := os.WriteFile(destCSS, source, 0644); err != nil {
		app.serverError(w, fmt.Errorf("error writing CSS file: %v", err))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Project built successfully"))
}

func (app *application) generateProject(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Get project ID from URL params
	params := httprouter.ParamsFromContext(r.Context())
	projectID := params.ByName("id")

	// For now, we'll use the sample project data
	projectData, err := os.ReadFile("data/sample_project.json")
	if err != nil {
		app.serverError(w, err)
		return
	}

	var project models.Project
	if err := json.Unmarshal(projectData, &project); err != nil {
		app.serverError(w, err)
		return
	}

	// Initialize services
	templateService, err := services.NewTemplateService()
	if err != nil {
		app.serverError(w, err)
		return
	}

	cssCompiler, err := services.NewCSSCompiler()
	if err != nil {
		app.serverError(w, err)
		return
	}
	defer cssCompiler.Cleanup()

	// Generate HTML
	htmlContent, err := templateService.GenerateHTML(project)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Compile CSS based on the generated HTML
	cssContent, err := cssCompiler.Compile(htmlContent)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Create a buffer to hold the zip file
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	// Add index.html to the zip
	htmlWriter, err := zipWriter.Create("index.html")
	if err != nil {
		app.serverError(w, fmt.Errorf("error creating HTML file in zip: %v", err))
		return
	}
	if _, err := htmlWriter.Write(htmlContent); err != nil {
		app.serverError(w, fmt.Errorf("error writing HTML to zip: %v", err))
		return
	}

	// Add CSS directory and file to the zip
	cssWriter, err := zipWriter.Create("css/tailwind.css")
	if err != nil {
		app.serverError(w, fmt.Errorf("error creating CSS file in zip: %v", err))
		return
	}
	if _, err := cssWriter.Write(cssContent); err != nil {
		app.serverError(w, fmt.Errorf("error writing CSS to zip: %v", err))
		return
	}

	// Close the zip writer
	if err := zipWriter.Close(); err != nil {
		app.serverError(w, fmt.Errorf("error closing zip writer: %v", err))
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", projectID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", zipBuffer.Len()))

	// Write the zip file to the response
	if _, err := w.Write(zipBuffer.Bytes()); err != nil {
		app.serverError(w, fmt.Errorf("error writing zip to response: %v", err))
		return
	}

	// Log API time
	duration := time.Since(startTime)
	app.infoLog.Printf("Generate project API completed in %v", duration)
}

func (app *application) submitBuildJob(w http.ResponseWriter, r *http.Request) {
	// For now, we'll use the sample project data
	projectData, err := os.ReadFile("data/sample_project.json")
	if err != nil {
		app.serverError(w, err)
		return
	}

	var project models.Project
	if err := json.Unmarshal(projectData, &project); err != nil {
		app.serverError(w, err)
		return
	}

	// Submit job to queue
	jobID := app.jobQueue.SubmitJob(project)

	// Return job ID and WebSocket URL to client
	response := struct {
		JobID     string `json:"jobId"`
		SocketURL string `json:"socketUrl"`
	}{
		JobID:     jobID,
		SocketURL: fmt.Sprintf("/ws?jobId=%s", jobID),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		app.serverError(w, err)
		return
	}
}

func (app *application) getJobStatus(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	jobID := params.ByName("id")

	buildJob, err := app.jobQueue.GetJobStatus(jobID)
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	// If job is completed, include download URL
	response := struct {
		Status    string `json:"status"`
		Progress  int    `json:"progress"`
		Message   string `json:"message"`
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updatedAt"`
		Download  string `json:"download,omitempty"`
	}{
		Status:    string(buildJob.Status),
		Progress:  buildJob.Progress,
		Message:   buildJob.Message,
		CreatedAt: buildJob.CreatedAt.Format(time.RFC3339),
		UpdatedAt: buildJob.UpdatedAt.Format(time.RFC3339),
	}

	if buildJob.Status == job.StatusCompleted {
		response.Download = fmt.Sprintf("/jobs/%s/download", jobID)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		app.serverError(w, err)
		return
	}
}

func (app *application) downloadJobResult(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	jobID := params.ByName("id")

	buildJob, err := app.jobQueue.GetJobStatus(jobID)
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	if buildJob.Status != job.StatusCompleted || buildJob.Result == nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", jobID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buildJob.Result.ZipFile)))

	// Write the zip file to the response
	if _, err := w.Write(buildJob.Result.ZipFile); err != nil {
		app.serverError(w, err)
		return
	}
}

func (app *application) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	app.socketManager.HandleConnection(w, r)
}
