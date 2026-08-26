# Solar Farm Maintenance Backend

This directory contains the Go module and service entrypoint.

```bash
go test ./...
go build ./...
go run .
```

The `main` package starts the HTTP service on `PORT` (default `8080`). It exposes `GET /healthz`, `GET /api/maintenance-tasks`, `POST /api/maintenance-tasks/{id}/status`, `/`, and `/app.js`.
