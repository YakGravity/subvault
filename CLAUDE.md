# SubVault - Claude Code Instructions

## About

SubVault is a self-hosted subscription tracking application, originally based on [subtrackr](https://github.com/bscott/subtrackr) by Brian Scott. Licensed under AGPL-3.0.

## Development

### Prerequisites

- Go 1.21+
- SQLite3

### Running locally

```bash
PORT=23457 DATABASE_PATH=./data/subvault.db go run ./cmd/server/
```

### Building

```bash
go build -o subvault ./cmd/server/
```

### Docker

```bash
docker build -t subvault .
docker run -p 23457:23457 -v subvault-data:/app/data subvault
```

## Project Structure

- `cmd/server/` - Application entry point
- `internal/handler/` - HTTP handlers (settings, subscription, API)
- `internal/service/` - Business logic (interfaces in `interfaces.go`)
- `internal/database/` - SQLite database layer
- `internal/models/` - Data models
- `internal/i18n/` - Internationalization (EN/DE)
- `static/` - CSS, JS (HTMX 1.9.10 self-hosted), images
- `templates/` - Go HTML templates

## Code Style

- Go templates compiled at startup (restart server after template changes)
- HTMX: Use `onclick` + `htmx.ajax()`, not `hx-get`/`hx-trigger` on container divs
- CSRF: gorilla/csrf with Gin adapter
- Logging: `slog` throughout
- Error handling: Generic messages to client, details in slog only
- Design system: Custom CSS (`design-system.css`, `themes.css`), no Tailwind

## Git Commit Guidelines

- Use conventional commit format
- Keep messages clear and descriptive
- Reference issue numbers where applicable
