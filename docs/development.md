# Development

## Prerequisites

- Go 1.24+
- SQLite3 (or GCC for CGO)

## Local Development

```bash
go mod download
go run ./cmd/server/
```

The server starts on port 8080 by default. Templates are compiled at startup — restart the server after template changes.

## Building

### Binary

```bash
go build -o subvault ./cmd/server/
```

### Docker (multi-arch)

```bash
docker buildx build --platform linux/amd64,linux/arm64 -f docker/Dockerfile .
```

### Build from Source

```bash
git clone https://github.com/YakGravity/subvault.git
cd subvault
go build -o subvault ./cmd/server/
./subvault
```

## Project Structure

```
cmd/server/          Entry point, routing, dependency wiring
internal/
  handlers/          HTTP handlers (auth, settings, subscription, API)
  service/           Business logic layer
  repository/        Database access layer
  database/          SQLite initialization and migrations
  models/            Data models
  middleware/        Auth, CSRF, i18n middleware
  i18n/              Internationalization (locales in locales/)
web/
  static/            CSS, JS, fonts, images
  templates/         Go HTML templates (auth/, settings/, subscription/, partials/)
docker/              Dockerfile, docker-compose.yml
docs/                Documentation
```

## Code Style

- **HTMX**: Use `onclick` + `htmx.ajax()`, not `hx-get`/`hx-trigger` on container divs
- **Templates**: Go HTML templates compiled at startup (restart server after changes)
- **Logging**: `slog` throughout the codebase
- **Error handling**: Generic messages to the client, details only in slog
- **CSS**: Custom design system (`design-system.css`, `themes.css`), no Tailwind
- **CSRF**: gorilla/csrf with Gin adapter
