# Go Press Server

A Go-based server for generating and building web projects with real-time progress tracking and automatic cleanup.

## Features

- Project build submission
- Real-time progress tracking via WebSocket
- CSS compilation with Tailwind CSS
- HTML template generation
- Build result download
- Automatic job cleanup (30-minute expiration)
- Job availability checking

## Prerequisites

- Go 1.16 or later
- Node.js 20 or later

## Installation

1. Clone the repository:

```bash
git clone https://github.com/sawthetphyoe/go-press-server.git
cd go-press-server
```

2. Install Go dependencies:

```bash
go mod download
```

3. Install Node.js dependencies for CSS compilation:

```bash
cd internal/services/css/shared
npm install tailwindcss @tailwindcss/typography
```

## Project Structure

```
go-press-server/
├── cmd/
│   └── web/              # Main application entry point
├── internal/
│   ├── models/           # Data models
│   ├── services/
│   │   ├── css/         # CSS compilation service
│   │   ├── job/         # Job queue and processing
│   │   └── websocket/   # WebSocket management
│   └── templates/        # HTML templates
├── client/              # Test client
└── ui/                  # Static assets
```

## Running the Server

1. Start the main server:

```bash
go run cmd/web/main.go
```

The server will start on `http://localhost:4000`

2. In a separate terminal, start the test client:

```bash
cd client
go run server.go
```

The client will be available at `http://localhost:3000`

## API Endpoints

- `POST /projects/:id/build` - Submit a new build job
  - Returns: `{ jobId: string, socketUrl: string }`
- `GET /jobs/:id/check` - Check job and build folder availability
  - Returns: `{ exists: boolean, status: string, folderExists: boolean, expiresAt: string }`
- `GET /jobs/:id/download` - Download build result
- `GET /ws` - WebSocket connection for real-time updates

## Job Management

- Jobs expire after 30 minutes
- Automatic cleanup of expired jobs and their files
- Real-time progress tracking via WebSocket
- Build results available for download until expiration

## Test Client Usage

1. Open `http://localhost:3000` in your browser
2. Click "Start Build" to begin the build process
3. Watch the real-time progress updates
4. Check job availability before downloading
5. Download the result when the build is complete

## Development

### Adding New Features

1. Create new handlers in `cmd/web/handlers.go`
2. Add routes in `cmd/web/routers.go`
3. Implement services in `internal/services/`

### Job Cleanup

The server automatically cleans up:

- Jobs older than 30 minutes
- Associated build files and directories
- Memory resources for completed jobs

### Error Handling

The server provides appropriate HTTP status codes:

- 404: Job not found or expired
- 400: Job exists but not ready for download
- 410: Job files have been cleaned up
- 500: Server error
