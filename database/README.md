# GateKey Database

This directory contains the database schema and documentation for GateKey's PostgreSQL database.

## Requirements

- **PostgreSQL 16+** (tested with 16.10)
- Required extensions:
  - `pgcrypto` - Cryptographic functions for secure password hashing
  - `uuid-ossp` - UUID generation for primary keys

## Quick Start

### 1. Create Database and User

```sql
-- Connect as postgres superuser
CREATE USER gatekey WITH PASSWORD 'your-secure-password';
CREATE DATABASE gatekey OWNER gatekey;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE gatekey TO gatekey;
```

### 2. Enable Required Extensions

```sql
-- Connect to gatekey database
\c gatekey

-- Enable extensions (requires superuser or extension creation privileges)
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

### 3. Apply Schema

**Option A: Using migrations (recommended for production)**

GateKey uses [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations. The server automatically applies migrations on startup.

```bash
# Migrations are in /migrations directory
# The server applies them automatically when DATABASE_URL is configured
```

**Option B: Using schema dump (for development/testing)**

```bash
psql -U gatekey -d gatekey -f database/schema.sql
```

## Configuration

GateKey connects to PostgreSQL using a connection URL. Configure via environment variable:

```bash
# Basic connection
export DATABASE_URL="postgres://gatekey:password@localhost:5432/gatekey?sslmode=require"

# With SSL/TLS verification
export DATABASE_URL="postgres://gatekey:password@db.example.com:5432/gatekey?sslmode=verify-full&sslrootcert=/path/to/ca.crt"
```

### SSL/TLS Modes

| Mode | Description |
|------|-------------|
| `disable` | No SSL (not recommended for production) |
| `require` | SSL required, no certificate verification |
| `verify-ca` | SSL required, verify server certificate against CA |
| `verify-full` | SSL required, verify certificate and hostname |

**Production recommendation:** Use `verify-full` with proper CA certificates.

### Connection Pool Settings

Configure in `gatex.yaml`:

```yaml
database:
  driver: "postgres"
  url: "${DATABASE_URL}"
  max_open_conns: 25      # Maximum open connections
  max_idle_conns: 5       # Maximum idle connections
  conn_max_lifetime: "5m" # Connection lifetime
  ssl_mode: "${DATABASE_SSL_MODE}"
```

## Schema Overview

### Core Tables

| Table | Description |
|-------|-------------|
| `users` | User accounts (local and SSO) |
| `sessions` | Active user sessions |
| `api_keys` | API key authentication |
| `gateways` | OpenVPN gateway configurations |
| `wireguard_gateways` | WireGuard gateway configurations |
| `wireguard_configs` | WireGuard client configurations |
| `wireguard_peers` | WireGuard peer assignments |

### Access Control

| Table | Description |
|-------|-------------|
| `access_rules` | Network access rules (CIDR, IP, hostname) |
| `user_access_rules` | User-specific access assignments |
| `group_access_rules` | Group-based access assignments |
| `local_groups` | Local group definitions |
| `local_group_members` | Group membership |
| `geo_fence_rules` | Geographic access restrictions |

### Mesh Networking

| Table | Description |
|-------|-------------|
| `mesh_hubs` | Mesh network hub configurations |
| `mesh_clients` | Mesh network client enrollments |
| `mesh_hub_networks` | Networks accessible via mesh |

### PKI & Security

| Table | Description |
|-------|-------------|
| `pki_ca` | Certificate Authority storage |
| `certificate_revocations` | Revoked certificates |
| `login_logs` | Authentication audit logs |

### System

| Table | Description |
|-------|-------------|
| `system_settings` | System-wide configuration |
| `generated_configs` | Gateway configuration cache |
| `schema_migrations` | Migration version tracking |

## Migrations

Migrations are located in the `/migrations` directory and follow the naming convention:

```
000001_initial_schema.up.sql
000001_initial_schema.down.sql
```

### Running Migrations Manually

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Apply migrations
migrate -path ./migrations -database "$DATABASE_URL" up

# Rollback last migration
migrate -path ./migrations -database "$DATABASE_URL" down 1

# Check migration version
migrate -path ./migrations -database "$DATABASE_URL" version
```

## Backup & Restore

### Backup

```bash
# Schema only
pg_dump -U gatekey -d gatekey --schema-only > schema_backup.sql

# Full backup (schema + data)
pg_dump -U gatekey -d gatekey > full_backup.sql

# Compressed backup
pg_dump -U gatekey -d gatekey | gzip > backup_$(date +%Y%m%d).sql.gz
```

### Restore

```bash
# Restore from backup
psql -U gatekey -d gatekey < full_backup.sql

# Restore compressed backup
gunzip -c backup_20260111.sql.gz | psql -U gatekey -d gatekey
```

## Kubernetes Deployment

For Kubernetes deployments, use a StatefulSet with persistent storage:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgresql-secrets
type: Opaque
stringData:
  postgres-user: gatekey
  postgres-password: your-secure-password
  postgres-db: gatekey
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgresql
spec:
  serviceName: postgresql
  replicas: 1
  selector:
    matchLabels:
      app: postgresql
  template:
    spec:
      containers:
      - name: postgresql
        image: postgres:16-alpine
        env:
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: postgresql-secrets
              key: postgres-user
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgresql-secrets
              key: postgres-password
        - name: POSTGRES_DB
          valueFrom:
            secretKeyRef:
              name: postgresql-secrets
              key: postgres-db
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

## Troubleshooting

### Connection Issues

```bash
# Test connection
psql -U gatekey -h localhost -d gatekey -c "SELECT 1"

# Check PostgreSQL logs
docker logs postgres-container
# or
journalctl -u postgresql
```

### Extension Issues

```sql
-- Check installed extensions
SELECT * FROM pg_extension;

-- Verify pgcrypto is working
SELECT crypt('test', gen_salt('bf'));

-- Verify uuid-ossp is working
SELECT uuid_generate_v4();
```

### Migration Issues

```bash
# Force migration version (use with caution)
migrate -path ./migrations -database "$DATABASE_URL" force VERSION

# Check dirty state
migrate -path ./migrations -database "$DATABASE_URL" version
```

## Security Recommendations

1. **Use strong passwords** - Generate with: `openssl rand -base64 32`
2. **Enable SSL/TLS** - Use `sslmode=verify-full` in production
3. **Restrict network access** - Use firewall rules or Kubernetes NetworkPolicies
4. **Regular backups** - Implement automated backup schedules
5. **Audit logging** - Enable PostgreSQL audit logging for compliance
6. **Encrypt at rest** - Use encrypted storage volumes
