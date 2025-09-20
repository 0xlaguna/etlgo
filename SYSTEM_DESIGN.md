# System Design - ETL Service

## 🏗️ Architecture Overview

The ETL Service is designed as a modern, scalable, and maintainable system following clean architecture principles. It processes marketing data from multiple sources to generate business insights.

### High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   External      │    │      ETL        │    │   External      │
│   Ads API       │───▶│     Service     │───▶│   Sink API      │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   CRM API       │
                       │                 │
                       └─────────────────┘
```

## 🔄 Idempotency & Reprocessing

### Idempotency Strategy

**Problem**: ETL processes must be safe to re-run without creating duplicate data or inconsistent state.

**Solution**: 
- **Date-based Filtering**: The `since` parameter allows reprocessing from a specific date
- **In-Memory Deduplication**: Data is stored by date, allowing overwrites of existing data
- **Atomic Operations**: Each ETL run is treated as a complete replacement for the processed date range
- **Request Correlation**: Each ETL run has a unique request ID for tracing

**Implementation**:
```go
// Data is stored by date key, allowing safe overwrites
dateKey := ad.Date.Format("2006-01-02")
r.adsData[dateKey] = append(r.adsData[dateKey], ad)
```

**Benefits**:
- Safe to re-run failed ETL jobs
- No duplicate data concerns
- Clear audit trail with request IDs

## 📊 Data Partitioning & Retention

### Partitioning Strategy

**Current Implementation**: Date-based partitioning
- Data is partitioned by date (YYYY-MM-DD format)
- Each date partition contains all records for that day
- Enables efficient date-range queries

**Future Considerations**:
- **Time-based Partitions**: Could extend to hourly partitions for high-volume data
- **UTM-based Sharding**: Partition by UTM combinations for better query performance
- **Database Migration**: Move from in-memory to persistent storage (PostgreSQL, ClickHouse)

### Retention Policy

**Current**: In-memory storage (data lost on restart)
**Recommended**: 
- **Hot Data**: Last 90 days in fast storage
- **Warm Data**: 90-365 days in slower storage
- **Cold Data**: Archive older data

**Implementation Strategy**:
```go
// Future retention implementation
func (r *Repository) CleanupOldData(retentionDays int) {
    cutoff := time.Now().AddDate(0, 0, -retentionDays)
    for dateKey := range r.data {
        if date, _ := time.Parse("2006-01-02", dateKey); date.Before(cutoff) {
            delete(r.data, dateKey)
        }
    }
}
```

## ⚡ Concurrency & Throughput

### Worker Pool Architecture

**Design**: Configurable worker pools for concurrent processing

```go
// ETL Service Configuration
type ETLService struct {
    workerPool int  // Default: 10 workers
    batchSize  int  // Default: 100 records per batch
}
```

**Concurrency Levels**:

1. **Data Extraction**: Parallel API calls to Ads and CRM endpoints
   ```go
   go func() { adsData, adsErr = s.apiClient.FetchAdsData(ctx) }()
   go func() { crmData, crmErr = s.apiClient.FetchCRMData(ctx) }()
   ```

2. **Data Loading**: Parallel storage operations
   ```go
   go func() { adsErr = s.adRepo.Store(ctx, ads) }()
   go func() { crmErr = s.crmRepo.Store(ctx, opportunities) }()
   ```

3. **Metrics Calculation**: Worker pool for UTM-based processing
   ```go
   for i := 0; i < s.workerPool; i++ {
       go func() {
           for utm := range jobs {
               metric := s.calculateMetricForUTM(adsByUTM[utm], oppsByUTM[utm], utm)
               results <- metric
           }
       }()
   }
   ```

### Rate Limiting & Backpressure

**External API Protection**:
```go
rateLimiter: *rate.NewLimiter(rate.Limit(100), 10) // 100 req/sec, burst 10
```

**Benefits**:
- Prevents API rate limit violations
- Provides backpressure under load
- Configurable per API endpoint

## 🔍 Data Quality & UTM Handling

### UTM Normalization Strategy

**Problem**: Inconsistent or missing UTM parameters across data sources.

**Solution**: Comprehensive normalization with fallback values

```go
// UTM Normalization
utmCampaign := ad.UTMCampaign
if utmCampaign == "" {
    utmCampaign = "unknown"
}
```

**UTM Correlation Logic**:
1. **Primary**: Match by exact UTM combination (campaign + source + medium)
2. **Fallback**: Match by campaign only if source/medium missing
3. **Default**: Group unmatched data under "unknown" UTMs

**Data Quality Measures**:
- **Validation**: Date format validation with error logging
- **Sanitization**: Empty string normalization
- **Monitoring**: Track data quality metrics (missing UTMs, parse errors)
- **Alerting**: Alert on high error rates

### Error Handling Strategy

**Graceful Degradation**:
- Individual record failures don't stop the entire ETL process
- Failed records are logged and counted in metrics
- Partial data processing is preferred over complete failure

**Error Categories**:
- **Network Errors**: Retry with exponential backoff
- **Data Quality**: Log and skip invalid records
- **System Errors**: Fail fast with clear error messages

## 📈 Observability & Monitoring

### Structured Logging

**Format**: JSON with correlation IDs
```json
{
  "timestamp": "2025-01-01T10:00:00Z",
  "level": "info",
  "message": "ETL pipeline completed successfully",
  "request_id": "uuid",
  "duration": "2.5s",
  "ads_records": 1000,
  "crm_records": 500
}
```

**Key Log Events**:
- ETL pipeline start/completion
- External API calls (success/failure)
- Data quality issues
- Performance metrics

### Prometheus Metrics

**HTTP Metrics**:
- Request count by endpoint and status code
- Request duration histograms
- Requests in flight

**ETL Metrics**:
- Job success/failure rates
- Processing duration
- Records processed/failed

**Business Metrics**:
- Metrics calculation counts
- Data quality indicators

**External API Metrics**:
- Call counts and durations
- Failure rates by error type
- Rate limiting events

### Health Checks

**Liveness Probe** (`/health`):
- Basic service health
- Memory usage
- Goroutine count

## 🚀 Evolution

### Data Lake Integration

**Current State**: In-memory processing
**Future Vision**: Data lake integration

**Migration Path**:
1. **Phase 1**: Add persistent storage (PostgreSQL)
2. **Phase 2**: Implement data lake connectors (S3, BigQuery)
3. **Phase 3**: Real-time streaming (Kafka, Kinesis)

**Benefits**:
- Historical data retention
- Cross-service data sharing
- Advanced analytics capabilities

### API Contract Evolution

**Versioning Strategy**:
- **URL Versioning**: `/api/v1/`, `/api/v2/`
- **Backward Compatibility**: Maintain v1 for 6 months
- **Deprecation Process**: Clear migration path and timeline

**Contract Management**:
```go
// API Versioning
type APIVersion struct {
    Version     string `json:"version"`
    Deprecated  bool   `json:"deprecated"`
    SunsetDate  string `json:"sunset_date,omitempty"`
}
```

### Microservices Architecture

**Current**: Monolithic service
**Future**: Microservices decomposition

**Potential Services**:
- **ETL Service**: Data processing and transformation
- **Metrics Service**: Business metrics calculation
- **Export Service**: Data export and delivery
- **API Gateway**: Request routing and authentication

**Communication Patterns**:
- **Synchronous**: HTTP/REST for real-time queries
- **Asynchronous**: Message queues for ETL events
- **Event Sourcing**: Audit trail and replay capability

### Scalability Considerations

**Horizontal Scaling**:
- **Stateless Design**: No shared state between instances
- **Load Balancing**: Multiple service instances
- **Database Sharding**: Partition data across multiple databases

**Vertical Scaling**:
- **Resource Optimization**: CPU/memory tuning
- **Caching**: Redis for frequently accessed data
- **Connection Pooling**: Efficient database connections

**Add a background job framework**:
- **Asynq**
- **RiverQueue**

**Auto-scaling Triggers**:
- CPU utilization > 70%
- Memory usage > 80%
- Request queue depth > 100
- ETL job backlog > 10 minutes

## 🔒 Security Considerations

### Data Protection

**In Transit**:
- HTTPS for all external communications
- TLS 1.3 for API endpoints
- Certificate pinning for external APIs

**At Rest**:
- Encryption for sensitive data
- Secure key management
- Data anonymization for logs

### Access Control

**API Authentication**:
- JWT tokens for API access
- Role-based access control (RBAC)
- Rate limiting per user/API key

**Audit Logging**:
- All API calls logged with user context
- Data access patterns tracked
- Compliance reporting capabilities

### Optimization Opportunities

**Database Optimization**:
- Indexes on date and UTM fields
- Partitioning for large datasets
- Query optimization and caching

**Caching Strategy**:
- **Application Cache**: Frequently accessed metrics
- **CDN**: Static API responses
- **Database Cache**: Query result caching

**Resource Optimization**:
- **Memory**: Efficient data structures
- **CPU**: Parallel processing optimization
- **Network**: Connection pooling and compression
