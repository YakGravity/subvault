# SubVault

A self-hosted subscription management application built with Go and HTMX. Track your subscriptions, visualize spending, and get renewal reminders.

## Features

- **Dashboard Overview**: Real-time stats showing monthly/annual spending
- **Subscription Management**: Track all your subscriptions in one place with logos
- **Calendar View**: Visual calendar showing all subscription renewal dates with iCal export
- **Analytics**: Visualize spending by category and track savings
- **Email & Pushover Notifications**: Get reminders before subscriptions renew
- **Data Export**: Export your data as CSV, JSON, iCal, or encrypted backup format
- **12 Themes**: 6 color palettes with light/dark mode, accent colors, sidebar collapse
- **Multi-Currency Support**: 30+ currencies with automatic ECB exchange rates (no API key needed)
- **Docker Ready**: Production-ready Docker image with non-root user and healthcheck
- **Self-Hosted**: Your data stays on your server
- **Mobile Responsive**: Optimized mobile experience
- **REST API**: Full API with key-based authentication
- **i18n**: English and German

## Tech Stack

- **Backend**: Go with Gin framework
- **Database**: SQLite (no external database needed)
- **Frontend**: HTMX + custom CSS design system (no Tailwind)
- **Deployment**: Docker & Docker Compose

## Quick Start

### Docker Compose (Recommended)

```yaml
services:
  subvault:
    image: ghcr.io/YakGravity/subvault:latest
    ports:
      - "8080:8080"
    volumes:
      - subvault_data:/app/data
    environment:
      - GIN_MODE=release
      - DATABASE_PATH=/app/data/subvault.db
    restart: unless-stopped

volumes:
  subvault_data:
    driver: local
```

```bash
docker compose up -d
```

Open http://localhost:8080.

### Build from Source

```bash
git clone https://github.com/YakGravity/subvault.git
cd subvault

# Build and run
make build
./subvault

# Or with Docker
make docker-build && make docker-up
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DATABASE_PATH` | SQLite database file path | `./data/subvault.db` |
| `GIN_MODE` | Gin framework mode (debug/release) | `debug` |

### Currency Conversion

SubVault provides automatic currency conversion using European Central Bank (ECB) exchange rates - no API key required.

### Notifications

- **Email (SMTP)**: Configure via Settings > Email Notifications
- **Pushover**: Configure via Settings > Pushover Notifications

### Data Persistence

Always mount a volume to `/app/data` to persist your database.

## Migration from SubTrackr

If you're upgrading from SubTrackr:

1. **Database**: Rename `subtrackr.db` to `subvault.db` (or update `DATABASE_PATH`)
2. **Docker volumes**: Update volume names in your compose file
3. **localStorage**: Theme preferences will reset (new key names)
4. **Session**: Users will need to log in again

## API Documentation

SubVault provides a RESTful API for external integrations. Create an API key from Settings in the web interface.

```bash
# Authorization header
curl -H "Authorization: Bearer sk_your_api_key_here" http://localhost:8080/api/v1/subscriptions
```

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/subscriptions` | List all subscriptions |
| POST | `/api/v1/subscriptions` | Create a new subscription |
| GET | `/api/v1/subscriptions/:id` | Get subscription details |
| PUT | `/api/v1/subscriptions/:id` | Update subscription |
| DELETE | `/api/v1/subscriptions/:id` | Delete subscription |
| GET | `/api/v1/stats` | Get subscription statistics |
| GET | `/api/v1/export/csv` | Export as CSV |
| GET | `/api/v1/export/json` | Export as JSON |
| GET | `/api/v1/export/ical` | Export as iCal |
| GET | `/api/v1/categories` | List all categories |
| POST | `/api/v1/categories` | Create a new category |
| PUT | `/api/v1/categories/:id` | Update category |
| DELETE | `/api/v1/categories/:id` | Delete category |

## Development

### Prerequisites

- Go 1.24+
- Docker (optional)

### Local Development

```bash
go mod download
go run cmd/server/main.go

# Build binary
make build
```

## Attribution

SubVault is a fork of [SubTrackr](https://github.com/bscott/subtrackr) by [bscott](https://github.com/bscott), originally created as a self-hosted subscription tracker. This fork extends the original with additional features including multi-currency support, extended theme system, i18n, encrypted backups, and a production-ready Docker setup.

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) - see the [LICENSE](LICENSE) file for details.
