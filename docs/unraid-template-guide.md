# Unraid Community Applications: Docker-Image + Template erstellen

A guide based on our experience building the SubVault Unraid template.

## Overview

Three parts:
1. **Docker image** with PUID/PGID support
2. **CA template XML** for Unraid
3. **Template repo** on GitHub for CA indexing

---

## 1. Docker Image (Unraid-compatible)

### Dockerfile — Key Elements

```dockerfile
FROM debian:bookworm-slim

# gosu for privilege dropping (industry standard for Debian)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gosu curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/myapp .
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN mkdir -p /app/data && chmod +x /app/entrypoint.sh

EXPOSE 8080

# PUID/PGID defaults: 99:100 = nobody:users (Unraid standard)
ENV PUID=99
ENV PGID=100

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

# No USER directive! Container starts as root, entrypoint drops privileges
ENTRYPOINT ["/app/entrypoint.sh"]
```

### Entrypoint Script (`docker/entrypoint.sh`)

```sh
#!/bin/sh
set -e

PUID=${PUID:-99}
PGID=${PGID:-100}

echo "Starting with PUID=$PUID PGID=$PGID"

# Create group if it doesn't exist
if ! getent group "$PGID" > /dev/null 2>&1; then
    addgroup --gid "$PGID" appuser
fi

# Create user if it doesn't exist
if ! getent passwd "$PUID" > /dev/null 2>&1; then
    GROUP_NAME=$(getent group "$PGID" | cut -d: -f1)
    adduser --uid "$PUID" --ingroup "$GROUP_NAME" \
            --disabled-password --no-create-home --gecos "" appuser
fi

# Fix ownership of data directory
chown -R "$PUID:$PGID" /app/data

# Drop privileges and run the application
exec gosu "$PUID:$PGID" ./myapp
```

### Why gosu?

| Tool | Base | Available in |
|------|------|-------------|
| **gosu** | Go binary | Debian (`apt install gosu`) |
| su-exec | C | Alpine (`apk add su-exec`) — **NOT in Debian!** |
| s6-overlay | Init system | LinuxServer.io — overkill for simple apps |

### Image on GitHub Container Registry (ghcr.io)

GitHub Actions workflow (`.github/workflows/docker-publish.yml`):
- Trigger: `push tags: ['v*']`
- Multi-arch: `linux/amd64,linux/arm64`
- Tags: `latest` + semver (`v1.3.2`)

---

## 2. CA Template XML

### File: `unraid/myapp.xml`

```xml
<?xml version="1.0"?>
<Container version="2">
  <Name>myapp</Name>
  <Repository>ghcr.io/user/myapp:latest</Repository>
  <Registry>https://github.com/user/myapp/pkgs/container/myapp</Registry>
  <Branch>
    <Tag>latest</Tag>
    <TagDescription>Latest stable release</TagDescription>
  </Branch>
  <Network>bridge</Network>
  <Privileged>false</Privileged>
  <Support>https://github.com/user/myapp/issues</Support>
  <Project>https://github.com/user/myapp</Project>
  <Overview>Short description of the app (shown in CA)</Overview>
  <Category>Productivity: Finance</Category>
  <WebUI>http://[IP]:[PORT:8080]/</WebUI>
  <Icon>https://raw.githubusercontent.com/user/myapp/main/web/static/images/icon.png</Icon>
  <Screenshot>https://raw.githubusercontent.com/user/myapp/main/screenshots/screenshot.png</Screenshot>
  <ExtraSearchTerms>search terms for CA</ExtraSearchTerms>
  <Shell>bash</Shell>
  <Changes>
### v1.0.0
- Initial release
  </Changes>

  <!-- Port -->
  <Config Name="Web Port" Target="8080" Default="8080" Mode="tcp"
    Description="Port for web interface" Type="Port"
    Display="always" Required="true" Mask="false">8080</Config>

  <!-- Volume -->
  <Config Name="Data Path" Target="/app/data" Default="/mnt/user/appdata/myapp"
    Description="SQLite database and app data" Type="Path"
    Display="always" Required="true" Mask="false">/mnt/user/appdata/myapp</Config>

  <!-- PUID/PGID -->
  <Config Name="PUID" Target="PUID" Default="99"
    Description="User ID for file permissions" Type="Variable"
    Display="always" Required="false" Mask="false">99</Config>

  <Config Name="PGID" Target="PGID" Default="100"
    Description="Group ID for file permissions" Type="Variable"
    Display="always" Required="false" Mask="false">100</Config>

  <!-- Timezone -->
  <Config Name="TZ" Target="TZ" Default="Europe/Berlin"
    Description="Container timezone" Type="Variable"
    Display="always" Required="false" Mask="false">Europe/Berlin</Config>
</Container>
```

### Config Types

| Type | Description | Example |
|------|-------------|---------|
| `Port` | Port mapping | `8080` |
| `Path` | Volume mount | `/mnt/user/appdata/...` |
| `Variable` | Environment variable | `PUID`, `TZ` |

### Display Values

| Display | Meaning |
|---------|---------|
| `always` | Always visible during installation |
| `advanced` | Only in Advanced View |

---

## 3. Template Repo for CA

### Repo Structure

```
github.com/user/unraid-templates/
├── templates/
│   └── myapp.xml          ← The CA template
└── README.md
```

### How Users Add the Repo in Unraid

1. Unraid WebUI → **Docker** → **Template Repositories**
2. Enter URL: `https://github.com/user/unraid-templates`
3. **Save** → Template appears under "Add Container"

### How to Get Listed in the Official CA App Store

1. Create a **forum thread** in [Docker Containers](https://forums.unraid.net/forum/54-docker-containers/)
2. **PM Squidly271** with link to the template repo
3. He adds the repo to the CA index

---

## Checklist

- [ ] Dockerfile with `gosu` + entrypoint (no `USER` directive)
- [ ] `PUID`/`PGID` env defaults set to `99`/`100`
- [ ] `HEALTHCHECK` in Dockerfile
- [ ] Image on ghcr.io (public, multi-arch amd64+arm64)
- [ ] CA template XML with Port, Volume, PUID/PGID, TZ
- [ ] `WebUI` with `[IP]:[PORT:xxxx]` syntax
- [ ] Icon URL on GitHub raw content
- [ ] Template repo on GitHub (public)
- [ ] Forum thread + PM to Squidly271
