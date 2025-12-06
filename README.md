# Juki Engine

**Juki Engine** is the backend service and command-line interface (CLI) runner for Juki Builder. It is responsible for data persistence, project management, and executing system commands required for the Next.js application lifecycle.

## Overview

The engine is written in Go and provides a RESTful API for the Juki Editor. It handles:
- **Project Storage**: Saving and retrieving project data (SQLite/PostgreSQL).
- **CLI Execution**: Running `npm` and `pnpm` commands for the user's application.
- **File System Operations**: Managing the generated code on the disk.

## Technology Stack

- **Language**: Go (1.23+)
- **Web Framework**: Echo v4
- **ORM**: GORM
- **API Framework**: Huma v2
- **Database**: SQLite (default) / PostgreSQL

## Setup & Development

To run the engine independently:

1.  Navigate to the engine directory:
    ```bash
    cd .juki/engine
    ```

2.  Run the server:
    ```bash
    go run cmd/main.go
    ```
    The server will start on the configured port (defaulting to 8080 or as specified in `.env`).

## API

The engine exposes endpoints for:
- `GET /projects`: List all projects.
- `POST /projects`: Create a new project.
- `GET /projects/:id`: Get project details.
- `PUT /projects/:id`: Update project state.
