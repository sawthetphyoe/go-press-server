package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"sawthet.go-press-server.net/internal/services/job"
	"sawthet.go-press-server.net/internal/services/websocket"
)

type application struct {
	infoLog       *log.Logger
	errorLog      *log.Logger
	jobQueue      *job.JobQueue
	socketManager *websocket.SocketManager
}

// CORS middleware
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Allow specific methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		// Allow specific headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Define command-line flags
	addr := flag.String("addr", ":4000", "HTTP network address")

	flag.Parse()

	// Create new loggers for the application
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	// Initialize job queue with 2 workers
	jobQueue := job.NewJobQueue(2)

	// Initialize WebSocket manager
	socketManager := websocket.NewSocketManager(jobQueue)

	// Initialize a new instance of application
	app := &application{
		infoLog:       infoLog,
		errorLog:      errorLog,
		jobQueue:      jobQueue,
		socketManager: socketManager,
	}

	// TLS Config is not needed for local development
	// tlsConfig := &tls.Config{
	// 	CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	// 	CipherSuites: []uint16{
	// 		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	// 		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	// 		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	// 		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	// 		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	// 		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	// 	},
	// 	MinVersion: tls.VersionTLS12,
	// 	MaxVersion: tls.VersionTLS13,
	// }

	// Create a new HTTP server
	srv := &http.Server{
		Addr:     *addr,
		ErrorLog: errorLog,
		Handler:  cors(app.routes()),
		// TLSConfig:    tlsConfig,
		IdleTimeout: time.Minute,
		ReadTimeout: 5 * time.Second,
		// TODO: Need to configure short write timeout with socket server implementation
		WriteTimeout: 30 * time.Second,
	}

	// Start the server and listen for requests
	infoLog.Printf("Starting server on %s", *addr)
	// err := srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	err := srv.ListenAndServe()
	errorLog.Fatal(err)
}
