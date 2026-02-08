# Changelog

All notable changes to SubVault will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.3.1] - 2026-02-08

### Fixed
- Wallos import now supports real Wallos export format (nested objects for currency, category, payment method)
- Wallos import handles both float and string price values

### Added
- Wallos import extracts start_date field

## [v1.3.0] - 2026-02-08

### Fixed
- Import FOREIGN KEY constraint error on fresh databases (#9)
- Category display on Settings > Data page (paginated API response handling)
- New category input layout in subscription form modal
- Three UI and localization issues (#4, #5, #7)

### Added
- Demo data file (35 sample subscriptions, 8 categories, 5 currencies)
- Application screenshots (Light + Dark themes)

### Changed
- Restructured README into dedicated docs directory

## [v1.2.0] - 2026-02-07

### Added
- Public API v1 with categories and iCal export endpoints
- API best practices: error handling, validation, pagination, CORS, rate limiting
- iCal colors (RFC 7986) and calendar event projection
- iCal all-day events, Trial/Paused support, cancellation dates

### Changed
- Docker files consolidated into `docker/` directory
- Templates reorganized into `web/templates/` with subdirectories
- Project structure cleanup

### Removed
- Outdated screenshots, Playwright setup and legacy files

## [v1.1.0] - 2026-02-06

### Changed
- **Rebrand from SubTrackr to SubVault**
- All references updated throughout codebase
- Import compatibility maintained for SubTrackr exports

## [v1.0.0] - 2026-02-05

### Added
- First stable release of SubVault
- Subscription tracking with multiple billing schedules
- 12 themes (6 palettes x Light/Dark)
- i18n support (EN/DE)
- CSRF protection and authentication
- Public API v1
- iCal calendar export
- Exchange rate management
- Email and Shoutrrr notifications

## [v0.19.0] - 2026-02-04

### Added
- Exchange rate status display
- Exchange rate fallback mechanism
- Exchange rate configuration options

### Fixed
- Mutex consistency in CurrencyService

## [v0.18.0] - 2026-02-03

### Fixed
- Sorting bug
- Currency settings display
- Template improvements

## [v0.17.0] - 2026-02-02

### Changed
- Move renewal date logic from GORM hooks to RenewalService

## [v0.16.0] - 2026-02-01

### Changed
- Split SettingsService into domain-specific services

## [v0.15.0] - 2026-01-31

### Added
- CSRF middleware with gorilla/csrf
- HTMX self-hosted (no CDN dependency)

### Fixed
- Session security improvements
- Modal fix
- Codebase audit quick-fixes

### Changed
- Sanitize error messages in HTTP responses
- i18n calendar month translations
- Cleanup orphaned i18n keys

## [v0.14.0] - 2026-01-30

### Changed
- Settings page cleanup and reorganization
- Sidebar collapse fix
- API docs restructuring

## [v0.13.0] - 2026-01-29

### Changed
- Settings and dashboard consistency improvements
- Handler split into domain-specific files
- Service interfaces extraction
- Migrate logging to slog

## [v0.12.0] - 2026-01-28

### Added
- Appearance settings page
- Template migration to new structure

## [v0.11.1] - 2026-01-27

### Fixed
- Table view bugs
- Clickable column headers for sorting

## [v0.11.0] - 2026-01-26

### Changed
- Complete UI redesign "Warm Utility"
- New design language (Sessions 4+5)

## [v0.10.0] - 2026-01-25

### Changed
- Comprehensive performance optimization

[v1.3.1]: https://github.com/YakGravity/subvault/compare/v1.3.0...v1.3.1
[v1.3.0]: https://github.com/YakGravity/subvault/compare/v1.2.0...v1.3.0
[v1.2.0]: https://github.com/YakGravity/subvault/compare/v1.1.0...v1.2.0
[v1.1.0]: https://github.com/YakGravity/subvault/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/YakGravity/subvault/compare/v0.19.0...v1.0.0
[v0.19.0]: https://github.com/YakGravity/subvault/compare/v0.18.0...v0.19.0
[v0.18.0]: https://github.com/YakGravity/subvault/compare/v0.17.0...v0.18.0
[v0.17.0]: https://github.com/YakGravity/subvault/compare/v0.16.0...v0.17.0
[v0.16.0]: https://github.com/YakGravity/subvault/compare/v0.15.0...v0.16.0
[v0.15.0]: https://github.com/YakGravity/subvault/compare/v0.14.0...v0.15.0
[v0.14.0]: https://github.com/YakGravity/subvault/compare/v0.13.0...v0.14.0
[v0.13.0]: https://github.com/YakGravity/subvault/compare/v0.12.0...v0.13.0
[v0.12.0]: https://github.com/YakGravity/subvault/compare/v0.11.1...v0.12.0
[v0.11.1]: https://github.com/YakGravity/subvault/compare/v0.11.0...v0.11.1
[v0.11.0]: https://github.com/YakGravity/subvault/compare/v0.10.0...v0.11.0
[v0.10.0]: https://github.com/YakGravity/subvault/compare/v0.9.0...v0.10.0
