package job

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sawthet.go-press-server.net/internal/models"
	"sawthet.go-press-server.net/internal/services"
	"sawthet.go-press-server.net/internal/utils"
)

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

type BuildJob struct {
	ID        string
	Project   models.Project
	Status    JobStatus
	Progress  int
	Message   string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
	Result    *BuildResult
}

type BuildResult struct {
	ZipFile []byte
	Error   error
}

type JobQueue struct {
	jobs           map[string]*BuildJob
	jobsMux        sync.RWMutex
	workers        int
	workChan       chan *BuildJob
	ctx            context.Context
	cancel         context.CancelFunc
	cleanupRunning bool
	infoLog        *utils.ColoredLogger
	errorLog       *utils.ColoredLogger
}

func NewJobQueue(workers int, infoLog, errorLog *utils.ColoredLogger) *JobQueue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &JobQueue{
		jobs:     make(map[string]*BuildJob),
		workers:  workers,
		workChan: make(chan *BuildJob),
		ctx:      ctx,
		cancel:   cancel,
		infoLog:  infoLog,
		errorLog: errorLog,
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		go q.worker()
	}

	return q
}

func (q *JobQueue) SubmitJob(project models.Project) string {
	if project.ID == "" {
		q.errorLog.Printf("Project ID is required")
		return ""
	}

	// Check if job exists
	q.jobsMux.Lock()
	existingJob, exists := q.jobs[project.ID]
	if exists {
		// If job is running, return error
		if existingJob.Status == StatusRunning {
			q.jobsMux.Unlock()
			q.errorLog.Printf("Project %s is already being built", project.ID)
			return ""
		}

		// Clean up old job files if they exist
		jobDir := filepath.Join("static", "sites", project.ID)
		if err := os.RemoveAll(jobDir); err != nil {
			q.errorLog.Printf("Failed to cleanup old job directory %s: %v", jobDir, err)
		}
	}
	q.jobsMux.Unlock()

	job := &BuildJob{
		ID:        project.ID,
		Project:   project,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	q.jobsMux.Lock()
	q.jobs[job.ID] = job
	q.jobsMux.Unlock()

	if !q.cleanupRunning {
		q.cleanupRunning = true
		go q.startCleanupRoutine()
	}

	// Send to work channel
	q.workChan <- job

	return job.ID
}

func (q *JobQueue) GetJobStatus(jobID string) (*BuildJob, error) {
	q.jobsMux.RLock()
	defer q.jobsMux.RUnlock()

	job, exists := q.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}

	return job, nil
}

func (q *JobQueue) worker() {
	for job := range q.workChan {
		q.processJob(job)
	}
}

func (q *JobQueue) processJob(job *BuildJob) {
	q.updateJobStatus(job, StatusRunning, 0, "Starting build process")

	// Initialize services
	templateService, err := services.NewTemplateService()
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 0, fmt.Sprintf("Failed to initialize template service: %v", err))
		return
	}

	cssCompiler, err := services.NewCSSCompiler()
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 0, fmt.Sprintf("Failed to initialize CSS compiler: %v", err))
		return
	}
	defer cssCompiler.Cleanup()

	// Generate HTML
	q.updateJobStatus(job, StatusRunning, 25, "Generating HTML...")
	htmlContent, err := templateService.GenerateHTML(job.Project)
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 25, fmt.Sprintf("Failed to generate HTML: %v", err))
		return
	}

	// Compile CSS
	q.updateJobStatus(job, StatusRunning, 50, "Compiling CSS...")
	cssContent, err := cssCompiler.Compile(htmlContent)
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 50, fmt.Sprintf("Failed to compile CSS: %v", err))
		return
	}

	// Create sites directory if it doesn't exist
	sitesDir := filepath.Join("static", "sites")
	if err := os.MkdirAll(sitesDir, 0755); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create sites directory: %v", err))
		return
	}

	// Create zip file directly in sites directory
	zipPath := filepath.Join(sitesDir, job.ID+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create zip file: %v", err))
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add HTML file to zip
	htmlWriter, err := zipWriter.Create("index.html")
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create HTML entry in zip: %v", err))
		return
	}
	if _, err := htmlWriter.Write(htmlContent); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to write HTML to zip: %v", err))
		return
	}

	// Add CSS directory and file to zip
	cssWriter, err := zipWriter.Create("css/tailwind.css")
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create CSS entry in zip: %v", err))
		return
	}
	if _, err := cssWriter.Write(cssContent); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to write CSS to zip: %v", err))
		return
	}

	// Update job status
	q.updateJobStatus(job, StatusCompleted, 100, "Build completed successfully!")
}

func (q *JobQueue) updateJobStatus(job *BuildJob, status JobStatus, progress int, message string) {
	q.jobsMux.Lock()
	defer q.jobsMux.Unlock()

	job.Status = status
	job.Progress = progress
	job.Message = message
	job.UpdatedAt = time.Now()
}

// startCleanupRoutine runs a background routine to clean up expired jobs
func (q *JobQueue) startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		// Get the next expiration time
		nextExpiration := q.getNextExpirationTime()

		if nextExpiration.IsZero() {
			q.cleanupRunning = false
			return
		}

		// Calculate sleep duration until next expiration
		now := time.Now()

		if nextExpiration.After(now) {
			sleepDuration := nextExpiration.Sub(now)

			select {
			case <-time.After(sleepDuration):
				q.cleanupExpiredJobs()
			case <-ticker.C:
				q.cleanupExpiredJobs()
			case <-q.ctx.Done():
				q.cleanupRunning = false
				return
			}
		} else {
			q.cleanupExpiredJobs()
		}
	}
}

// getNextExpirationTime returns the earliest expiration time among all jobs
func (q *JobQueue) getNextExpirationTime() time.Time {
	q.jobsMux.RLock()
	defer q.jobsMux.RUnlock()

	if len(q.jobs) == 0 {
		return time.Time{}
	}

	var nextExpiration time.Time
	for _, job := range q.jobs {
		if nextExpiration.IsZero() || job.ExpiresAt.Before(nextExpiration) {
			nextExpiration = job.ExpiresAt
		}
	}
	return nextExpiration
}

// cleanupExpiredJobs removes jobs that have expired
func (q *JobQueue) cleanupExpiredJobs() {
	q.jobsMux.Lock()
	defer q.jobsMux.Unlock()

	now := time.Now()

	for id, job := range q.jobs {
		if now.After(job.ExpiresAt) {
			// Remove job from memory
			delete(q.jobs, id)

			// Remove zip file from disk
			zipPath := filepath.Join("static", "sites", id+".zip")
			if err := os.Remove(zipPath); err != nil {
				q.errorLog.Printf("Failed to cleanup zip file %s: %v", zipPath, err)
			}
		}
	}
}

// CleanupAllJobs stops the cleanup routine and removes all jobs
func (q *JobQueue) CleanupAllJobs() {
	// Stop the cleanup routine
	q.cancel()

	// Remove all zip files
	sitesDir := filepath.Join("static", "sites")
	if err := os.RemoveAll(sitesDir); err != nil {
		q.errorLog.Printf("Failed to cleanup sites directory: %v", err)
	}

	// Clear all jobs from memory
	q.jobsMux.Lock()
	q.jobs = make(map[string]*BuildJob)
	q.jobsMux.Unlock()
}
