# SubVault

[![Go](https://img.shields.io/badge/go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![Lizenz: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-green)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ghcr.io-2496ED?logo=docker&logoColor=white)](https://ghcr.io/yakgravity/subvault)
[![Build](https://github.com/YakGravity/subvault/actions/workflows/test-build.yml/badge.svg)](https://github.com/YakGravity/subvault/actions/workflows/test-build.yml)
[![Built with Claude](https://img.shields.io/badge/built%20with-Claude%20Code-cc785c?logo=anthropic&logoColor=white)](https://claude.ai/claude-code)

Eine selbst gehostete Abo-Verwaltung, gebaut mit Go und HTMX. Verfolge wiederkehrende Ausgaben, visualisiere Analysen und erhalte Verlängerungserinnerungen — alles auf deinem eigenen Server.

> **[English version](README.md)**

## Screenshots

<details open>
<summary>Dashboard</summary>

![Dashboard Hell](screenshots/dashboard-light.png)
![Dashboard Dunkel](screenshots/dashboard-dark.png)
</details>

<details>
<summary>Abonnements</summary>

![Abonnements Kacheln Hell](screenshots/subscriptions-grid-light.png)
![Abonnements Liste Dunkel](screenshots/subscriptions-list-dark.png)
</details>

<details>
<summary>Kalender</summary>

![Kalender Hell](screenshots/calendar-light.png)
![Kalender Dunkel](screenshots/calendar-dark.png)
</details>

<details>
<summary>Einstellungen</summary>

![Einstellungen Hell](screenshots/settings-light.png)
![Einstellungen Dunkel](screenshots/settings-dark.png)
</details>

## Funktionen

- **Dashboard** — Monatliche/jährliche Ausgabenübersicht mit Kategorie-Aufschlüsselung
- **Abo-Verwaltung** — Alle wiederkehrenden Ausgaben verwalten mit automatischem Logo-Abruf
- **Kalenderansicht** — Visueller Verlängerungskalender mit iCal-Export (RFC 7986 Farben)
- **Analysen** — Ausgabentrends nach Kategorie, monatliche Projektionen, Sparübersicht
- **Benachrichtigungen** — E-Mail (SMTP) und Push-Benachrichtigungen via [Shoutrrr](https://containrrr.dev/shoutrrr/)
- **Multi-Währung** — 30+ Währungen mit automatischen Wechselkursen der [Europäischen Zentralbank](https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml) (täglicher XML-Feed, EUR-basierte Cross-Rates, lokal gecacht mit Offline-Fallback)
- **12 Themes** — 6 Farbpaletten mit Hell-/Dunkelmodus, Akzentfarben, einklappbare Seitenleiste
- **Datenexport** — CSV, JSON, iCal und verschlüsseltes Backup-Format
- **REST API** — Vollständige CRUD-API mit schlüsselbasierter Authentifizierung
- **i18n** — Englisch und Deutsch eingebaut, eigene Sprachen per Locale-Dateien
- **Mobil optimiert** — Für alle Bildschirmgrößen angepasst
- **Self-Hosted** — SQLite-Datenbank, keine externen Abhängigkeiten

## Schnellstart

```yaml
# docker-compose.yml
services:
  subvault:
    image: ghcr.io/yakgravity/subvault:latest
    ports:
      - "8080:8080"
    volumes:
      - subvault_data:/app/data
    restart: unless-stopped

volumes:
  subvault_data:
```

```bash
docker compose up -d
```

Öffne [http://localhost:8080](http://localhost:8080) — keine Ersteinrichtung erforderlich.

## Dokumentation

| Thema | Beschreibung |
|-------|-------------|
| [Konfiguration](docs/configuration.md) | Umgebungsvariablen, Docker-Setup, Benachrichtigungen, Reverse Proxy |
| [API](docs/api.md) | REST-API-Endpunkte, Authentifizierung, Beispiele |
| [Entwicklung](docs/development.md) | Lokales Setup, Build, Projektstruktur, Code-Style |
| [Migration](docs/migration.md) | Migration von SubTrackr |

## Tech Stack

| Komponente | Technologie |
|-----------|-----------|
| Backend | Go 1.24, Gin |
| Datenbank | SQLite (via GORM) |
| Frontend | HTMX, eigenes CSS-Design-System |
| Auth | Session-basiert + API-Keys, CSRF-Schutz |
| i18n | go-i18n (Englisch, Deutsch + eigene Locales) |
| Container | Multi-Arch Docker (amd64, arm64) |

## Mit KI entwickelt

Dieses Projekt wird hauptsächlich mit [Claude Code](https://claude.ai/claude-code) (Anthropics Claude) entwickelt. Architektur, Implementierung, Debugging und Dokumentation sind KI-generiert mit menschlicher Steuerung und Review. Jeder Commit ist von Claude co-authored — die Git-Historie zeigt volle Transparenz.

## Herkunft

SubVault ist ein Fork von [SubTrackr](https://github.com/bscott/subtrackr) von [Brian Scott](https://github.com/bscott). Dieser Fork erweitert das Original um Multi-Währungs-Unterstützung, ein erweitertes Theme-System, Internationalisierung, verschlüsselte Backups, Push-Benachrichtigungen, Kalender-Integration und ein produktionsreifes Docker-Setup.

## Lizenz

[GNU Affero General Public License v3.0 (AGPL-3.0)](LICENSE)
