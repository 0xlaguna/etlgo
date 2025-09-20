# ETL Service

## 🚀 Features

- **Concurrent Processing**: Worker pools for high-performance data processing
- **Business Metrics**: CPC, CPA, CVR, ROAS calculations with UTM correlation
- **REST API**: Comprehensive API with filtering, pagination, and export
- **Observability**: Structured logging, Prometheus metrics, health checks
- **Docker Ready**: Complete containerization with Docker Compose
- **Idempotent**: Safe to re-run ETL processes without data duplication
- **Rate Limiting**: Built-in rate limiting and retry logic with exponential backoff

## 📋 Requirements

- Go 1.25
- Docker & Docker Compose (for containerized deployment)

## 🛠️ Quick Start

### Using Docker Compose (Recommended)

1. **Clone and start the service:**
   ```bash
   git clone <repository-url>
   cd goetl
   docker compose up --build
   ```

2. **Clone and start the service and run with monitoring(Optional):**
   ```bash
   git clone <repository-url>
   cd goetl
   docker compose --profile monitoring up --build
   ```

2. **The service will be available at:**
   - API: http://localhost:8080
   - Health Check: http://localhost:8080/health
   - Metrics: http://localhost:8080/metrics


## 🔧 Configuration

It will load everything on env.example for demo purposes


### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ADS_API_URL` | Ads API endpoint | Required |
| `CRM_API_URL` | CRM API endpoint | Required |
| `SINK_URL` | Export destination URL | Optional |
| `SINK_SECRET` | HMAC secret for exports | Optional |
| `PORT` | Server port | 8080 |
| `LOG_LEVEL` | Logging level | info |
| `WORKER_POOL_SIZE` | ETL worker pool size | 10 |
| `BATCH_SIZE` | Processing batch size | 100 |
| `REQUEST_TIMEOUT` | HTTP request timeout | 30s |
| `MAX_RETRIES` | Max retry attempts | 3 |
| `RATE_LIMIT_PER_SECOND` | Rate limit per second | 100 |

## 📚 API Endpoints

### ETL Operations

#### Trigger ETL Pipeline
```bash
POST /api/v1/ingest/run?since=2025-01-01
```

**Parameters:**
- `since` (optional): Filter data from this date (YYYY-MM-DD format)

**Response:**
```json
{
  "message": "ETL ingestion completed successfully",
  "request_id": "uuid",
  "since": "2025-01-01"
}
```

### Metrics Queries

#### Get Metrics by Channel
```bash
GET /api/v1/metrics/channel?channel=facebook_ads&from=2025-01-01&to=2025-10-18&limit=50&offset=0
```

#### Get Metrics by Funnel (UTM Campaign)
```bash
GET /api/v1/metrics/funnel?utm_campaign=fall_sale&from=2025-01-01&to=2025-10-18
```

#### Get Metrics Summary
```bash
GET /api/v1/metrics/summary
```

**Response:**
```json
{
    "averages": {
        "cpa": 624.3100000000001,
        "cpc": 0.3495576707726764,
        "cvr_lead_to_opp": 1.2,
        "cvr_opp_to_won": 0.8333333333333334,
        "roas": 6.951674648812288
    },
    "counts": {
        "metric_records": 10,
        "unique_campaigns": 10,
        "unique_channels": 4
    },
    "period": {
        "from": "2025-07-22",
        "to": "2025-09-20"
    },
    "request_id": "349e144d-8717-4e78-a1c7-cef4be1c25ac",
    "totals": {
        "clicks": 8930,
        "closed_won": 5,
        "cost": 3121.55,
        "impressions": 383000,
        "leads": 5,
        "opportunities": 6,
        "revenue": 21700
    }
}
```

## 📊 Business Metrics

The service calculates the following business metrics:

- **CPC (Cost Per Click)**: `cost / clicks`
- **CPA (Cost Per Acquisition)**: `cost / leads`
- **CVR Lead to Opportunity**: `opportunities / leads`
- **CVR Opportunity to Won**: `closed_won / opportunities`
- **ROAS (Return on Ad Spend)**: `revenue / cost`

### Data Correlation

Data is correlated using UTM parameters:
- `utm_campaign`
- `utm_source`
- `utm_medium`

Missing UTM values are normalized to "unknown" for consistent processing.

## 🏗️ Architecture

### Clean Architecture Layers

```
cmd/server/          # Application entry point
internal/
├── domain/          # Business entities and interfaces
├── usecase/         # Business logic and orchestration
├── infrastructure/  # External dependencies (HTTP, storage)
└── delivery/        # HTTP handlers and routing
pkg/                 # Shared utilities
├── config/          # Configuration management
├── logger/          # Structured logging
└── metrics/         # Prometheus metrics
```

### Key Design Patterns

- **Dependency Injection**: All dependencies are injected for testability
- **Repository Pattern**: Abstract data access layer
- **Worker Pool Pattern**: Concurrent processing with configurable workers
- **Circuit Breaker**: Resilient external API calls
- **Rate Limiting**: Prevents API abuse

## 🚀 Performance Features

- **Concurrent Data Fetching**: Parallel API calls to Ads and CRM endpoints
- **Worker Pool Processing**: Configurable worker pools for metrics calculation
- **In-Memory Storage**: Fast data access with thread-safe operations
- **Connection Pooling**: Efficient HTTP client with connection reuse
- **Batch Processing**: Configurable batch sizes for optimal throughput

## 📈 Monitoring & Observability

### Structured Logging
- JSON format with request correlation IDs
- Configurable log levels
- Context-aware logging

### Prometheus Metrics
- HTTP request metrics (count, duration, status codes)
- ETL job metrics (success/failure rates, duration)
- External API metrics (call counts, failures, duration)
- Business metrics (calculation counts)

### Health Checks
- `/health`: Basic service health

## 🔒 Security Features

- **HMAC Signatures**: Secure export data with HMAC-SHA256
- **Request Timeouts**: Prevent resource exhaustion
- **Rate Limiting**: Protect against abuse
- **Non-root Container**: Security-hardened Docker image

## 🐳 Docker Deployment

### Basic Deployment
```bash
docker compose up --build
```

### With Monitoring Stack
```bash
docker compose --profile monitoring up --build
```

This starts:
- Admira ETL Service (port 8080)
- Prometheus (port 9090)
- Grafana (port 3000, admin/admin)


## 📝 Example Usage

### 1. Trigger ETL Pipeline
```bash
curl -X POST "http://localhost:8080/api/v1/ingest/run?since=2025-01-01"
```

### 2. Query Metrics by Channel
```bash
curl "http://localhost:8080/api/v1/metrics/channel?channel=facebook_ads&limit=50&offset=0"
```

### 3. Export Data
```bash
curl -X POST "http://localhost:8080/api/v1/export/run?date=2025-08-06"
```

### 4. Check Health
```bash
curl "http://localhost:8080/health"
```
