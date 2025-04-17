package websocket

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"sawthet.go-press-server.net/internal/services/job"
)

type SocketManager struct {
	clients    map[string]*websocket.Conn
	clientsMux sync.RWMutex
	jobQueue   *job.JobQueue
}

func NewSocketManager(jobQueue *job.JobQueue) *SocketManager {
	return &SocketManager{
		clients:  make(map[string]*websocket.Conn),
		jobQueue: jobQueue,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for WebSocket connections
		return true
	},
}

func (sm *SocketManager) HandleConnection(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("jobId")
	if jobID == "" {
		http.Error(w, "jobId is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	sm.clientsMux.Lock()
	sm.clients[jobID] = conn
	sm.clientsMux.Unlock()

	go sm.monitorJobProgress(jobID, conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			sm.clientsMux.Lock()
			delete(sm.clients, jobID)
			sm.clientsMux.Unlock()
			break
		}
	}
}

func (sm *SocketManager) monitorJobProgress(jobID string, conn *websocket.Conn) {
	buildJob, err := sm.jobQueue.GetJobStatus(jobID)
	if err != nil {
		sm.sendProgress(conn, jobID, string(job.StatusFailed), 0, fmt.Sprintf("Error: %v", err))
		sm.cleanupConnection(jobID, conn)
		return
	}

	// If job is already completed or failed, send final status and return
	if buildJob.Status == job.StatusCompleted || buildJob.Status == job.StatusFailed {
		sm.sendProgress(conn, jobID, string(buildJob.Status), buildJob.Progress, buildJob.Message)
		sm.cleanupConnection(jobID, conn)
		return
	}

	// Monitor progress updates
	for {
		select {
		case progress := <-buildJob.ProgressChan:
			sm.sendProgress(conn, jobID, string(progress.Status), progress.Progress, progress.Message)

			if progress.Status == job.StatusCompleted || progress.Status == job.StatusFailed {
				sm.cleanupConnection(jobID, conn)
				return
			}
		case <-time.After(30 * time.Second):
			sm.sendProgress(conn, jobID, string(job.StatusFailed), 0, "Connection timeout")
			sm.cleanupConnection(jobID, conn)
			return
		}
	}
}

func (sm *SocketManager) sendProgress(conn *websocket.Conn, jobID, status string, progress int, message string) {
	msg := struct {
		JobID    string `json:"jobId"`
		Status   string `json:"status"`
		Progress int    `json:"progress"`
		Message  string `json:"message"`
	}{
		JobID:    jobID,
		Status:   status,
		Progress: progress,
		Message:  message,
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

// cleanupConnection closes the WebSocket connection and removes it from the clients map
func (sm *SocketManager) cleanupConnection(jobID string, conn *websocket.Conn) {
	// Send close message
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Job completed"))

	// Close the connection
	conn.Close()

	// Remove from clients map
	sm.clientsMux.Lock()
	delete(sm.clients, jobID)
	sm.clientsMux.Unlock()
}

// Cleanup closes all active WebSocket connections
func (sm *SocketManager) Cleanup() {
	sm.clientsMux.Lock()
	defer sm.clientsMux.Unlock()

	for jobID, conn := range sm.clients {
		// Send close message
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutting down"))

		// Close connection
		conn.Close()

		// Remove from map
		delete(sm.clients, jobID)
	}
}
