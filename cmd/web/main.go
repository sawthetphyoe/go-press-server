package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sawthet.go-press-server.net/internal/services"
	"sawthet.go-press-server.net/internal/services/job"
	"sawthet.go-press-server.net/internal/services/websocket"
	"sawthet.go-press-server.net/internal/utils"
)

type application struct {
	infoLog         *utils.ColoredLogger
	errorLog        *utils.ColoredLogger
	jobQueue        *job.JobQueue
	socketManager   *websocket.SocketManager
	templateService *services.TemplateService
	cssCompiler     *services.CSSCompiler
}

func main() {
	// Initialize loggers
	infoLog := utils.NewColoredLogger("INFO", "\033[32m")   // Green color for info
	errorLog := utils.NewColoredLogger("ERROR", "\033[31m") // Red color for error

	// Initialize job queue with 2 workers
	jobQueue := job.NewJobQueue(2, infoLog, errorLog)

	// Initialize WebSocket manager
	socketManager := websocket.NewSocketManager(jobQueue)

	// Initialize services
	templateService, err := services.NewTemplateService()
	if err != nil {
		errorLog.Printf("Failed to initialize template service: %v", err)
		os.Exit(1)
	}

	cssCompiler, err := services.NewCSSCompiler()
	if err != nil {
		errorLog.Printf("Failed to initialize CSS compiler: %v", err)
		os.Exit(1)
	}

	// Initialize application
	app := &application{
		infoLog:         infoLog,
		errorLog:        errorLog,
		jobQueue:        jobQueue,
		socketManager:   socketManager,
		templateService: templateService,
		cssCompiler:     cssCompiler,
	}

	// Create server
	server := &http.Server{
		Addr:         ":4000",
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Channel to receive OS signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		infoLog.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorLog.Printf("Server error: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-done
	infoLog.Println("Server is gracefully shutting down...")

	// Cleanup all jobs
	jobQueue.CleanupAllJobs()

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		errorLog.Printf("Server forced to shutdown: %v", err)
	}

	infoLog.Println("Server exited properly")
}
