# Configuration

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DATABASE_PATH` | SQLite database file path | `./data/subvault.db` |
| `GIN_MODE` | `debug` or `release` | `debug` |
| `HTTPS_ENABLED` | Set to `true` behind a TLS-terminating reverse proxy | `false` |
| `LOCALE_DIR` | Directory for custom locale files | _(empty)_ |

## Custom Languages

SubVault ships with English and German built-in. You can add new languages or override existing translations by placing locale files in a directory and setting `LOCALE_DIR`.

**Adding a new language:**

1. Copy `docs/active.en.json` as a starting point
2. Rename it to `active.XX.json` (e.g., `active.fr.json` for French)
3. Translate the values (keys must stay the same)
4. Add a `"lang_self"` key with the native language name (e.g., `"Français"`)
5. Place the file in your locale directory and set `LOCALE_DIR`

Missing keys automatically fall back to English. You don't need to translate everything at once.

**Overriding built-in translations:**

Place an `active.en.json` or `active.de.json` in your `LOCALE_DIR` — it will override the built-in version.

**Docker example:**

```yaml
volumes:
  - ./my-locales:/app/locales
environment:
  - LOCALE_DIR=/app/locales
```

## Data Persistence

Always mount a volume to `/app/data` to persist your database. The SQLite database file contains all your subscriptions, settings, and API keys.

## Notifications

Configure via the web interface under **Settings > Notifications**:

- **Email (SMTP)** — Any SMTP provider (Gmail, Fastmail, self-hosted)
- **Push Notifications** — Via [Shoutrrr](https://containrrr.dev/shoutrrr/) supporting Pushover, Telegram, Discord, Slack, and more

## Reverse Proxy

SubVault works behind any reverse proxy (Nginx, Caddy, Traefik). Set `HTTPS_ENABLED=true` when using TLS termination so that CSRF cookies are configured correctly.

## Docker CLI

```bash
docker run -d \
  --name subvault \
  -p 8080:8080 \
  -v subvault_data:/app/data \
  -e GIN_MODE=release \
  --restart unless-stopped \
  ghcr.io/yakgravity/subvault:latest
```

## Docker Compose

```yaml
services:
  subvault:
    image: ghcr.io/yakgravity/subvault:latest
    ports:
      - "8080:8080"
    volumes:
      - subvault_data:/app/data
    environment:
      - GIN_MODE=release
    restart: unless-stopped

volumes:
  subvault_data:
```

```bash
docker compose up -d
```

Open [http://localhost:8080](http://localhost:8080) — no initial setup required.
