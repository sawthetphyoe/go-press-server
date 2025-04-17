package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/julienschmidt/httprouter"
	"sawthet.go-press-server.net/internal/models"
	"sawthet.go-press-server.net/internal/services/job"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
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

func (app *application) downloadJobResult(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	jobID := params.ByName("id")

	buildJob, err := app.jobQueue.GetJobStatus(jobID)
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	if buildJob.Status != job.StatusCompleted {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Check if job directory exists
	jobDir := filepath.Join("static", "sites", jobID)
	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		app.clientError(w, http.StatusGone)
		return
	}

	// Serve zip file from disk
	zipPath := filepath.Join(jobDir, "build.zip")
	zipFile, err := os.Open(zipPath)
	if err != nil {
		app.serverError(w, err)
		return
	}
	defer zipFile.Close()

	// Set headers for file download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", jobID))

	// Stream the file to the client
	if _, err := io.Copy(w, zipFile); err != nil {
		app.serverError(w, err)
		return
	}
}

func (app *application) checkJobAvailability(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	jobID := params.ByName("id")

	buildJob, err := app.jobQueue.GetJobStatus(jobID)
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	// Check if job directory exists
	jobDir := filepath.Join("static", "sites", jobID)
	_, dirErr := os.Stat(jobDir)

	response := struct {
		Exists       bool   `json:"exists"`
		Status       string `json:"status"`
		FolderExists bool   `json:"folderExists"`
		ExpiresAt    string `json:"expiresAt"`
	}{
		Exists:       true,
		Status:       string(buildJob.Status),
		FolderExists: dirErr == nil,
		ExpiresAt:    buildJob.ExpiresAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		app.serverError(w, err)
		return
	}
}

func (app *application) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	app.socketManager.HandleConnection(w, r)
}
