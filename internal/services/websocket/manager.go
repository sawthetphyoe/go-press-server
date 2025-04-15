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
	for {
		buildJob, err := sm.jobQueue.GetJobStatus(jobID)
		if err != nil {
			sm.sendProgress(conn, jobID, string(job.StatusFailed), 0, fmt.Sprintf("Error: %v", err))
			return
		}

		sm.sendProgress(conn, jobID, string(buildJob.Status), buildJob.Progress, buildJob.Message)

		if buildJob.Status == job.StatusCompleted || buildJob.Status == job.StatusFailed {
			return
		}

		time.Sleep(500 * time.Millisecond)
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
