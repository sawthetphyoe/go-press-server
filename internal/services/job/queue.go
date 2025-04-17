package job

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
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
	job := &BuildJob{
		ID:        fmt.Sprintf("job-%d", time.Now().UnixNano()),
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

	// Create base directories
	baseDir := filepath.Join("static", "sites")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create directories: %v", err))
		return
	}

	// Create job-specific output directory
	outputDir := filepath.Join(baseDir, job.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create output directory: %v", err))
		return
	}

	// Write HTML file
	htmlPath := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(htmlPath, htmlContent, 0644); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to write HTML file: %v", err))
		return
	}

	// Create CSS directory and write CSS file
	cssDir := filepath.Join(outputDir, "css")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create CSS directory: %v", err))
		return
	}

	cssPath := filepath.Join(cssDir, "tailwind.css")
	if err := os.WriteFile(cssPath, cssContent, 0644); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to write CSS file: %v", err))
		return
	}

	// Create zip file
	zipPath := filepath.Join(outputDir, "build.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create zip file: %v", err))
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add files to zip
	files := []struct {
		Name string
		Path string
	}{
		{"index.html", htmlPath},
		{"css/tailwind.css", cssPath},
	}

	for _, file := range files {
		f, err := os.Open(file.Path)
		if err != nil {
			q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to open file for zip: %v", err))
			return
		}
		defer f.Close()

		zipEntry, err := zipWriter.Create(file.Name)
		if err != nil {
			q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create zip entry: %v", err))
			return
		}

		if _, err := io.Copy(zipEntry, f); err != nil {
			q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to write to zip: %v", err))
			return
		}
	}

	// Update job status without storing zip file in memory
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

			// Remove job files from disk
			jobDir := filepath.Join("static", "sites", id)
			if err := os.RemoveAll(jobDir); err != nil {
				q.errorLog.Printf("Failed to cleanup job directory %s: %v", jobDir, err)
			}
		}
	}
}

// CleanupAllJobs stops the cleanup routine and removes all jobs
func (q *JobQueue) CleanupAllJobs() {
	// Stop the cleanup routine
	q.cancel()

	// Remove all job directories
	baseDir := filepath.Join("static", "sites")
	if err := os.RemoveAll(baseDir); err != nil {
		log.Printf("Failed to cleanup all job directories: %v", err)
	}

	// Clear all jobs from memory
	q.jobs = make(map[string]*BuildJob)
}
