package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	// Initialize a new httprouter instance
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	// Create a file server
	fileServer := http.FileServer(http.Dir("./ui/static"))

	// Handle static files
	router.Handler(http.MethodGet, "/static/*filepath", http.StripPrefix("/static", fileServer))

	// Home page endpoint
	router.HandlerFunc(http.MethodGet, "/", app.home)

	// Job endpoints
	router.HandlerFunc(http.MethodPost, "/projects/:id/build", app.submitBuildJob)
	router.HandlerFunc(http.MethodGet, "/jobs/:id", app.getJobStatus)
	router.HandlerFunc(http.MethodGet, "/jobs/:id/download", app.downloadJobResult)

	// WebSocket endpoint
	router.HandlerFunc(http.MethodGet, "/ws", app.handleWebSocket)

	standard := alice.New(app.recoverPanic, app.logRequest, app.secureHeaders)

	return standard.Then(router)
}
