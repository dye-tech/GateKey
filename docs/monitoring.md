# Login Monitoring

GateKey provides comprehensive login monitoring to track authentication events across your deployment. This feature helps with security auditing, compliance requirements, and identifying suspicious activity.

## Overview

The Login Monitoring feature captures all authentication events, including:
- Successful logins via OIDC, SAML, and local authentication
- Failed login attempts with failure reasons
- IP addresses and geolocation data
- User agent information

## Accessing the Monitoring Page

Navigate to **Administration → Monitoring** in the web UI. The page has three tabs:

1. **Login Logs** - View and filter all authentication events
2. **Statistics** - Aggregated metrics and charts
3. **Settings** - Configure log retention

## Login Logs

### Filtering Options

| Filter | Description |
|--------|-------------|
| Email | Search by user email (partial match) |
| IP Address | Search by client IP (partial match) |
| Provider | Filter by auth provider (OIDC, SAML, Local) |
| Status | Filter by success or failure |

### Data Fields

Each log entry contains:

| Field | Description |
|-------|-------------|
| `id` | Unique identifier (UUID) |
| `user_id` | User's unique identifier |
| `user_email` | User's email address |
| `user_name` | Display name (if available) |
| `provider` | Authentication type (oidc, saml, local) |
| `provider_name` | Specific provider name (e.g., "Okta") |
| `ip_address` | Client's public IP address |
| `user_agent` | Browser/client information |
| `country` | Geolocation country (if available) |
| `city` | Geolocation city (if available) |
| `success` | Whether login succeeded |
| `failure_reason` | Reason for failure (if applicable) |
| `session_id` | Associated session ID |
| `created_at` | Timestamp of the event |

## Statistics

The Statistics tab provides aggregated metrics for the past 30 days:

- **Total Logins** - All authentication attempts
- **Successful Logins** - Successful authentications
- **Failed Logins** - Failed authentication attempts
- **Unique Users** - Distinct users who logged in
- **Unique IPs** - Distinct IP addresses

### Charts

- **Logins by Provider** - Distribution across OIDC, SAML, Local
- **Logins by Country** - Top 10 countries by login count
- **Recent Failures** - Last 10 failed login attempts

## Log Retention

### Automatic Cleanup

A background job runs every 6 hours to delete logs older than the retention period.

### Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| Retention Days | 30 | Days to keep login logs |
| Value of 0 | Forever | Logs are never automatically deleted |

### Manual Purge

You can manually delete logs older than a specified number of days from the Settings tab. This action is irreversible.

## API Endpoints

### List Login Logs

```
GET /api/v1/admin/login-logs
```

Query Parameters:
- `user_email` - Filter by email (partial match)
- `user_id` - Filter by user ID (exact match)
- `ip_address` - Filter by IP (partial match)
- `provider` - Filter by provider (oidc, saml, local)
- `success` - Filter by success (true/false)
- `start_time` - Filter by start time (ISO 8601)
- `end_time` - Filter by end time (ISO 8601)
- `limit` - Results per page (default: 50)
- `offset` - Pagination offset

Response:
```json
{
  "logs": [...],
  "total": 1234
}
```

### Get Login Statistics

```
GET /api/v1/admin/login-logs/stats
```

Query Parameters:
- `days` - Number of days to include (default: 30)

Response:
```json
{
  "total_logins": 1000,
  "successful_logins": 950,
  "failed_logins": 50,
  "unique_users": 100,
  "unique_ips": 200,
  "logins_by_provider": {"oidc": 800, "saml": 150, "local": 50},
  "logins_by_country": {"United States": 600, "Canada": 200, ...},
  "recent_failures": [...]
}
```

### Purge Login Logs

```
DELETE /api/v1/admin/login-logs?days=30
```

Deletes all logs older than the specified number of days.

Response:
```json
{
  "deleted_count": 5000
}
```

### Get Retention Setting

```
GET /api/v1/admin/login-logs/retention
```

Response:
```json
{
  "days": 30
}
```

### Set Retention Setting

```
PUT /api/v1/admin/login-logs/retention
```

Request Body:
```json
{
  "days": 30
}
```

## Database Schema

The `login_logs` table stores all authentication events:

```sql
CREATE TABLE login_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    user_name VARCHAR(255),
    provider VARCHAR(50) NOT NULL,
    provider_name VARCHAR(100),
    ip_address INET NOT NULL,
    user_agent TEXT,
    country VARCHAR(100),
    city VARCHAR(100),
    success BOOLEAN NOT NULL DEFAULT true,
    failure_reason VARCHAR(255),
    session_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_login_logs_user_email ON login_logs(user_email);
CREATE INDEX idx_login_logs_user_id ON login_logs(user_id);
CREATE INDEX idx_login_logs_created_at ON login_logs(created_at DESC);
CREATE INDEX idx_login_logs_ip_address ON login_logs(ip_address);
CREATE INDEX idx_login_logs_success ON login_logs(success);
```

## Security Considerations

- Login logs contain PII (email, IP addresses) - ensure appropriate access controls
- Consider your compliance requirements when setting retention periods
- GDPR/CCPA may require the ability to delete user data on request
- Logs are stored in the same PostgreSQL database as other GateKey data

## Best Practices

1. **Set appropriate retention** - Balance security needs with storage costs
2. **Monitor failed logins** - Use the Statistics tab to identify brute force attempts
3. **Review unusual locations** - Watch for logins from unexpected countries
4. **Archive before purging** - Export logs before manual purge if needed for compliance
5. **Regular audits** - Periodically review login patterns for anomalies
