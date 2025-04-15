package job

import (
	"archive/zip"
	"bytes"
	"fmt"
	"sync"
	"time"

	"sawthet.go-press-server.net/internal/models"
	"sawthet.go-press-server.net/internal/services"
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
	Result    *BuildResult
}

type BuildResult struct {
	ZipFile []byte
	Error   error
}

type JobQueue struct {
	jobs     map[string]*BuildJob
	jobsMux  sync.RWMutex
	workers  int
	workChan chan *BuildJob
}

func NewJobQueue(workers int) *JobQueue {
	queue := &JobQueue{
		jobs:     make(map[string]*BuildJob),
		workChan: make(chan *BuildJob, 100),
		workers:  workers,
	}

	// Start worker pool
	for i := 0; i < workers; i++ {
		go queue.worker()
	}

	return queue
}

func (q *JobQueue) SubmitJob(project models.Project) string {
	job := &BuildJob{
		ID:        fmt.Sprintf("job-%d", time.Now().UnixNano()),
		Project:   project,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	q.jobsMux.Lock()
	q.jobs[job.ID] = job
	q.jobsMux.Unlock()

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
	q.updateJobStatus(job, StatusRunning, 25, "Generating HTML")
	htmlContent, err := templateService.GenerateHTML(job.Project)
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 25, fmt.Sprintf("Failed to generate HTML: %v", err))
		return
	}

	// Compile CSS
	q.updateJobStatus(job, StatusRunning, 50, "Compiling CSS")
	cssContent, err := cssCompiler.Compile(htmlContent)
	if err != nil {
		q.updateJobStatus(job, StatusFailed, 50, fmt.Sprintf("Failed to compile CSS: %v", err))
		return
	}

	// Create zip file
	q.updateJobStatus(job, StatusRunning, 75, "Creating zip file")
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	// Add files to zip
	if err := addFilesToZip(zipWriter, htmlContent, cssContent); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to create zip file: %v", err))
		return
	}

	// Close zip writer
	if err := zipWriter.Close(); err != nil {
		q.updateJobStatus(job, StatusFailed, 75, fmt.Sprintf("Failed to close zip file: %v", err))
		return
	}

	// Update job with result
	job.Result = &BuildResult{
		ZipFile: zipBuffer.Bytes(),
	}
	q.updateJobStatus(job, StatusCompleted, 100, "Build completed successfully")
}

func (q *JobQueue) updateJobStatus(job *BuildJob, status JobStatus, progress int, message string) {
	q.jobsMux.Lock()
	defer q.jobsMux.Unlock()

	job.Status = status
	job.Progress = progress
	job.Message = message
	job.UpdatedAt = time.Now()
}

func addFilesToZip(zipWriter *zip.Writer, htmlContent, cssContent []byte) error {
	// Add index.html
	htmlWriter, err := zipWriter.Create("index.html")
	if err != nil {
		return err
	}
	if _, err := htmlWriter.Write(htmlContent); err != nil {
		return err
	}

	// Add CSS directory and file
	cssWriter, err := zipWriter.Create("css/tailwind.css")
	if err != nil {
		return err
	}
	if _, err := cssWriter.Write(cssContent); err != nil {
		return err
	}

	return nil
}
