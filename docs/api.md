# API

SubVault provides a RESTful API for external integrations.

## Authentication

Create an API key from **Settings > API Keys** in the web interface. Pass it via the `Authorization` header:

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:8080/api/v1/subscriptions
```

## Endpoints

### Subscriptions

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/subscriptions` | List all subscriptions |
| `POST` | `/api/v1/subscriptions` | Create subscription |
| `GET` | `/api/v1/subscriptions/:id` | Get subscription |
| `PUT` | `/api/v1/subscriptions/:id` | Update subscription |
| `DELETE` | `/api/v1/subscriptions/:id` | Delete subscription |

### Categories

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/categories` | List categories |
| `POST` | `/api/v1/categories` | Create category |
| `PUT` | `/api/v1/categories/:id` | Update category |
| `DELETE` | `/api/v1/categories/:id` | Delete category |

### Statistics & Export

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/stats` | Spending statistics |
| `GET` | `/api/v1/export/csv` | Export as CSV |
| `GET` | `/api/v1/export/json` | Export as JSON |
| `GET` | `/api/v1/export/ical` | Export as iCal |

## Examples

### List all subscriptions

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:8080/api/v1/subscriptions
```

### Create a subscription

```bash
curl -X POST \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "Netflix", "cost": 13.99, "billing_cycle": "monthly"}' \
  http://localhost:8080/api/v1/subscriptions
```

### Export as CSV

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  -o subscriptions.csv \
  http://localhost:8080/api/v1/export/csv
```

## In-App Documentation

Full API documentation with request/response schemas is available in the web interface under **API Docs**.
