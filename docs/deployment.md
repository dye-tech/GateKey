# GateKey Deployment Guide

This guide covers deploying GateKey in various environments.

## Home-Lab K3s Deployment (ArgoCD + GitOps)

This section covers deploying GateKey to the home-lab K3s cluster using ArgoCD and GitOps.

### Prerequisites

- K3s cluster running with Istio ingress
- ArgoCD installed and configured
- Harbor registry accessible at `harbor.dye.tech`
- Cloudflare DNS configured for `gatekey.dye.tech`
- Keycloak running for OIDC authentication

### Quick Deploy

1. **Build and push images to Harbor:**

```bash
cd /home/jesse/Desktop/gatekey

# Login to Harbor (if not already logged in)
docker login harbor.dye.tech

# Build and push all images
./scripts/build-push.sh
```

2. **Apply Cloudflare DNS (if not already done):**

```bash
cd /home/jesse/Desktop/firestar/terraform/k3s/cloud-flare-dns
terraform apply
```

3. **Deploy via ArgoCD:**

The application will be automatically discovered by the ApplicationSet. To manually trigger:

```bash
# Check if the application is registered
argocd app get gatekey

# Force sync if needed
argocd app sync gatekey
```

4. **Verify deployment:**

```bash
# Check pods
kubectl get pods -n gatekey

# Check services
kubectl get svc -n gatekey

# Test the endpoint
curl -k https://gatekey.dye.tech/health
```

### GitOps Structure

```
firestar/gitops/
├── applications/gatekey/           # Base manifests
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── secrets.yaml
│   ├── postgresql.yaml
│   ├── deployment-server.yaml
│   ├── deployment-web.yaml
│   ├── services.yaml
│   ├── virtualservice.yaml
│   └── kustomization.yaml
└── clusters/home-lab/applications/gatekey/  # Cluster overlay
    ├── application.yaml          # ArgoCD Application
    └── kustomization.yaml        # Overlay with secrets
```

### URLs

- **Web UI**: https://gatekey.dye.tech
- **API**: https://gatekey.dye.tech/api/v1/
- **Health**: https://gatekey.dye.tech/health
- **Metrics**: https://gatekey.dye.tech/metrics
- **Downloads**: https://gatekey.dye.tech/downloads
- **Install Script**: https://gatekey.dye.tech/scripts/install-gateway.sh

### Gateway Installation via Script

The easiest way to install a gateway is using the install script:

```bash
curl -sSL https://gatekey.dye.tech/scripts/install-gateway.sh | sudo bash -s -- \
  --server https://gatekey.dye.tech \
  --token YOUR_GATEWAY_TOKEN \
  --name my-gateway
```

Or using the shorthand URL:
```bash
curl -sSL https://gatekey.dye.tech/install.sh | sudo bash -s -- \
  --server https://gatekey.dye.tech \
  --token YOUR_GATEWAY_TOKEN \
  --name my-gateway
```

The script will:
1. Detect your OS and architecture
2. Download the appropriate gateway binary
3. Install and configure OpenVPN
4. Register the gateway with the control plane
5. Set up systemd services

### Istio VirtualService Routing

The VirtualService (`virtualservice.yaml`) routes traffic between the frontend (gatekey-web) and backend (gatekey-server):

| Path Pattern | Destination | Description |
|--------------|-------------|-------------|
| `/api/*` | gatekey-server | REST API endpoints |
| `/proxy/*` | gatekey-server | Reverse proxy for web apps |
| `/scripts/*` | gatekey-server | Install scripts |
| `/downloads/*` | gatekey-server | Binary downloads |
| `/install.sh` | gatekey-server | Gateway install script (alias) |
| `/health` | gatekey-server | Health check |
| `/metrics` | gatekey-server | Prometheus metrics |
| `/*` (default) | gatekey-web | React frontend |

**Important**: When proxied applications make JavaScript requests to absolute paths (e.g., `/pve2/images/logo.png`), the VirtualService uses the Referer header to route these to gatekey-server, which redirects them to the correct proxy path.

### Keycloak OIDC Client

The GateKey OIDC client is configured in:
`/home/jesse/Desktop/firestar/gitops/infrastructure/keycloak/keycloak-clients-config.yaml`

Client ID: `gatekey`
Redirect URIs:
- `https://gatekey.dye.tech/api/v1/auth/oidc/callback`
- `https://gatekey.dye.tech/api/v1/auth/cli/callback`
- `http://localhost:*` (for CLI login)

### Troubleshooting K3s Deployment

**Check ArgoCD sync status:**
```bash
argocd app get gatekey
argocd app logs gatekey
```

**Check pod logs:**
```bash
kubectl logs -n gatekey -l app.kubernetes.io/component=server -f
kubectl logs -n gatekey -l app.kubernetes.io/component=web -f
```

**Check database:**
```bash
kubectl exec -it -n gatekey postgresql-0 -- psql -U gatekey -d gatekey
```

**Rebuild and redeploy:**
```bash
cd /home/jesse/Desktop/gatekey
./scripts/build-push.sh
kubectl rollout restart deployment -n gatekey gatekey-server gatekey-web
```

---

## Standard Deployment

## Prerequisites

- Go 1.23+
- PostgreSQL 14+
- OpenVPN 2.5+
- Node.js 20+ (for building frontend)
- Linux server with nftables support

## Quick Start

### 1. Build Binaries

```bash
# Clone the repository
git clone https://github.com/gatekey-project/gatekey.git
cd gatekey

# Install dependencies
make deps

# Build all binaries
make build
```

This produces:
- `bin/gatekey-server` - Control plane server
- `bin/gatekey-gateway` - Gateway agent
- `bin/gatekey` - CLI tool

### 2. Database Setup

```bash
# Create database
createdb gatekey

# Set connection string
export DATABASE_URL="postgres://user:pass@localhost/gatekey?sslmode=disable"

# Run migrations
make migrate-up
```

### 3. Configure Control Plane

Copy and edit the configuration:

```bash
mkdir -p /etc/gatekey
cp configs/gatekey.yaml /etc/gatekey/
```

Edit `/etc/gatekey/gatekey.yaml`:

```yaml
server:
  address: ":8080"
  tls_enabled: true
  tls_cert: "/etc/gatekey/certs/server.crt"
  tls_key: "/etc/gatekey/certs/server.key"

database:
  url: "postgres://gatekey:secret@localhost/gatekey?sslmode=require"

auth:
  oidc:
    enabled: true
    providers:
      - name: "default"
        display_name: "Login with SSO"
        issuer: "https://your-idp.example.com"
        client_id: "gatekey"
        client_secret: "your-secret"
        redirect_url: "https://gatekey.example.com/api/v1/auth/oidc/callback"
        scopes: ["openid", "profile", "email", "groups"]
```

### 4. Start Control Plane

```bash
# As a systemd service
sudo cp deploy/gatekey-server.service /etc/systemd/system/
sudo systemctl enable gatekey-server
sudo systemctl start gatekey-server

# Or directly
./bin/gatekey-server --config /etc/gatekey/gatekey.yaml
```

### 5. Build Frontend

```bash
cd web
npm install
npm run build

# Copy to web root
sudo cp -r dist/* /var/www/gatekey/
```

### 6. Configure Gateway

On each gateway server:

```bash
# Install OpenVPN
sudo apt install openvpn

# Copy gateway binary
sudo cp bin/gatekey-gateway /usr/local/bin/

# Create configuration
sudo mkdir -p /etc/gatekey
sudo cp configs/gateway.yaml /etc/gatekey/
```

Edit `/etc/gatekey/gateway.yaml`:

```yaml
control_plane_url: "https://gatekey.example.com"
token: "your-gateway-token"  # Generate via control plane

openvpn:
  status_file: "/var/log/openvpn/status.log"
  client_config_dir: "/etc/openvpn/ccd"

firewall:
  backend: "nftables"
```

### 7. Configure OpenVPN

Generate server configuration:

```bash
./bin/gatekey generate-server-config \
  --name "gateway-1" \
  --network "172.31.255.0/24" \
  --port 1194 \
  --proto udp \
  > /etc/openvpn/server.conf
```

Enable hooks in `/etc/openvpn/server.conf`:

```
auth-user-pass-verify /etc/openvpn/hooks/gatekey-hook.sh via-file
tls-verify /etc/openvpn/hooks/gatekey-hook.sh
client-connect /etc/openvpn/hooks/gatekey-hook.sh
client-disconnect /etc/openvpn/hooks/gatekey-hook.sh
script-security 2
```

### 8. Start Services

```bash
# Start gateway agent
sudo systemctl enable gatekey-gateway
sudo systemctl start gatekey-gateway

# Start OpenVPN
sudo systemctl enable openvpn@server
sudo systemctl start openvpn@server
```

## Production Deployment

### SSL/TLS Certificates

Use Let's Encrypt or your own CA:

```bash
# Using certbot
sudo certbot certonly --standalone -d gatekey.example.com

# Update config
server:
  tls_cert: "/etc/letsencrypt/live/gatekey.example.com/fullchain.pem"
  tls_key: "/etc/letsencrypt/live/gatekey.example.com/privkey.pem"
```

### Reverse Proxy (nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name gatekey.example.com;

    ssl_certificate /etc/letsencrypt/live/gatekey.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/gatekey.example.com/privkey.pem;

    # Frontend
    location / {
        root /var/www/gatekey;
        try_files $uri $uri/ /index.html;
    }

    # API
    location /api {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### High Availability

For HA deployment:

1. Run multiple control plane instances behind a load balancer
2. Use PostgreSQL with replication
3. Configure shared session storage (Redis)

### Monitoring

Enable Prometheus metrics:

```yaml
metrics:
  enabled: true
  path: "/metrics"
  port: 9090
```

Add to Prometheus config:

```yaml
scrape_configs:
  - job_name: 'gatekey'
    static_configs:
      - targets: ['gatekey.example.com:9090']
```

### Logging

Configure structured logging:

```yaml
logging:
  level: "info"
  format: "json"
  output: "/var/log/gatekey/server.log"
```

Use log rotation:

```bash
# /etc/logrotate.d/gatekey
/var/log/gatekey/*.log {
    daily
    rotate 14
    compress
    missingok
    notifempty
    create 0640 gatekey gatekey
    postrotate
        systemctl reload gatekey-server
    endscript
}
```

## Docker Deployment

### Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: gatekey
      POSTGRES_USER: gatekey
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data

  gatekey-server:
    build: .
    ports:
      - "8080:8080"
    environment:
      gatekey_DATABASE_URL: "postgres://gatekey:secret@postgres/gatekey?sslmode=disable"
    depends_on:
      - postgres

  gatekey-web:
    build: ./web
    ports:
      - "3000:80"

volumes:
  pgdata:
```

### Kubernetes

See `deploy/kubernetes/` for Helm charts and manifests.

## Troubleshooting

### Control Plane Won't Start

Check logs:
```bash
journalctl -u gatekey-server -f
```

Common issues:
- Database connection failed
- TLS certificate not found
- Port already in use

### Gateway Connection Failed

Check:
1. Gateway token is valid
2. Control plane is reachable
3. Firewall allows connections

### OpenVPN Hooks Failing

Check:
1. Hook scripts are executable
2. gatekey-gateway is running
3. Gateway can reach control plane

Debug hooks:
```bash
gatekey_LOG_LEVEL=debug gatekey-gateway hook --type auth-user-pass-verify
```
