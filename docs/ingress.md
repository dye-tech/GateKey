# Ingress Configuration

This guide covers configuring ingress for GateKey in Kubernetes environments. GateKey requires HTTP/HTTPS ingress for the web UI and API, plus UDP ingress for VPN traffic (OpenVPN or WireGuard).

## Overview

GateKey exposes the following services:

| Service | Protocol | Port | Purpose |
|---------|----------|------|---------|
| Web UI | HTTPS | 443 | Admin dashboard and user portal |
| API | HTTPS | 443 | REST API (`/api/*`) |
| OpenVPN | UDP | 1194 | VPN connections (configurable) |
| WireGuard | UDP | 51820 | VPN connections (configurable) |

## Istio Service Mesh

Istio is the recommended ingress solution for production deployments, providing mTLS, traffic management, and observability.

### Gateway Configuration

Create an Istio Gateway for HTTPS traffic:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: gatekey-gateway
  namespace: istio-system
spec:
  selector:
    istio: ingressgateway
  servers:
    - port:
        number: 443
        name: https
        protocol: HTTPS
      tls:
        mode: SIMPLE
        credentialName: gatekey-tls-cert
      hosts:
        - vpn.yourcompany.com
```

### VirtualService

Route traffic to the appropriate backend services:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: gatekey
  namespace: gatekey
spec:
  hosts:
    - vpn.yourcompany.com
  gateways:
    - istio-system/gatekey-gateway
  http:
    # API routes
    - match:
        - uri:
            prefix: /api/
      route:
        - destination:
            host: gatekey-server
            port:
              number: 8080
    # Binary downloads
    - match:
        - uri:
            prefix: /downloads/
      route:
        - destination:
            host: gatekey-server
            port:
              number: 8080
    - match:
        - uri:
            prefix: /bin/
      route:
        - destination:
            host: gatekey-server
            port:
              number: 8080
    # Install scripts
    - match:
        - uri:
            prefix: /scripts/
      route:
        - destination:
            host: gatekey-server
            port:
              number: 8080
    - match:
        - uri:
            exact: /install.sh
      route:
        - destination:
            host: gatekey-server
            port:
              number: 8080
    # Health and metrics
    - match:
        - uri:
            exact: /health
      route:
        - destination:
            host: gatekey-server
            port:
              number: 8080
    - match:
        - uri:
            exact: /metrics
      route:
        - destination:
            host: gatekey-server
            port:
              number: 8080
    # Proxy routes (for reverse proxy feature)
    - match:
        - uri:
            prefix: /proxy/
      route:
        - destination:
            host: gatekey-server
            port:
              number: 8080
    # Frontend (catch-all)
    - route:
        - destination:
            host: gatekey-web
            port:
              number: 80
```

### mTLS Configuration

Enable strict mTLS within the GateKey namespace:

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: gatekey
spec:
  mtls:
    mode: STRICT
```

### TLS Settings

Harden TLS configuration:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: gatekey-tls
  namespace: gatekey
spec:
  host: "*.gatekey.svc.cluster.local"
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
```

## NGINX Ingress Controller

NGINX Ingress is widely used and well-supported.

### Basic Configuration

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gatekey
  namespace: gatekey
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - vpn.yourcompany.com
      secretName: gatekey-tls
  rules:
    - host: vpn.yourcompany.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: gatekey-server
                port:
                  number: 8080
          - path: /downloads
            pathType: Prefix
            backend:
              service:
                name: gatekey-server
                port:
                  number: 8080
          - path: /bin
            pathType: Prefix
            backend:
              service:
                name: gatekey-server
                port:
                  number: 8080
          - path: /scripts
            pathType: Prefix
            backend:
              service:
                name: gatekey-server
                port:
                  number: 8080
          - path: /health
            pathType: Exact
            backend:
              service:
                name: gatekey-server
                port:
                  number: 8080
          - path: /metrics
            pathType: Exact
            backend:
              service:
                name: gatekey-server
                port:
                  number: 8080
          - path: /proxy
            pathType: Prefix
            backend:
              service:
                name: gatekey-server
                port:
                  number: 8080
          - path: /
            pathType: Prefix
            backend:
              service:
                name: gatekey-web
                port:
                  number: 80
```

### TLS Hardening

Add security-focused annotations:

```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/ssl-protocols: "TLSv1.3"
    nginx.ingress.kubernetes.io/ssl-ciphers: "ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384"
    nginx.ingress.kubernetes.io/ssl-prefer-server-ciphers: "true"
    nginx.ingress.kubernetes.io/hsts: "true"
    nginx.ingress.kubernetes.io/hsts-max-age: "31536000"
    nginx.ingress.kubernetes.io/hsts-include-subdomains: "true"
    nginx.ingress.kubernetes.io/hsts-preload: "true"
```

### Rate Limiting

Protect against abuse:

```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/limit-rps: "20"
    nginx.ingress.kubernetes.io/limit-connections: "10"
```

## Traefik

Traefik provides automatic service discovery and Let's Encrypt integration.

### IngressRoute

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: gatekey
  namespace: gatekey
spec:
  entryPoints:
    - websecure
  routes:
    # API routes
    - match: Host(`vpn.yourcompany.com`) && PathPrefix(`/api`)
      kind: Rule
      services:
        - name: gatekey-server
          port: 8080
    # Downloads
    - match: Host(`vpn.yourcompany.com`) && (PathPrefix(`/downloads`) || PathPrefix(`/bin`))
      kind: Rule
      services:
        - name: gatekey-server
          port: 8080
    # Scripts
    - match: Host(`vpn.yourcompany.com`) && PathPrefix(`/scripts`)
      kind: Rule
      services:
        - name: gatekey-server
          port: 8080
    # Proxy
    - match: Host(`vpn.yourcompany.com`) && PathPrefix(`/proxy`)
      kind: Rule
      services:
        - name: gatekey-server
          port: 8080
    # Health/Metrics
    - match: Host(`vpn.yourcompany.com`) && (Path(`/health`) || Path(`/metrics`))
      kind: Rule
      services:
        - name: gatekey-server
          port: 8080
    # Frontend (catch-all)
    - match: Host(`vpn.yourcompany.com`)
      kind: Rule
      services:
        - name: gatekey-web
          port: 80
  tls:
    secretName: gatekey-tls
```

### TLS Options

```yaml
apiVersion: traefik.io/v1alpha1
kind: TLSOption
metadata:
  name: default
  namespace: gatekey
spec:
  minVersion: VersionTLS13
  cipherSuites:
    - TLS_AES_256_GCM_SHA384
    - TLS_CHACHA20_POLY1305_SHA256
  sniStrict: true
```

### Middleware for Headers

```yaml
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: security-headers
  namespace: gatekey
spec:
  headers:
    stsSeconds: 31536000
    stsIncludeSubdomains: true
    stsPreload: true
    contentTypeNosniff: true
    browserXssFilter: true
    referrerPolicy: "strict-origin-when-cross-origin"
    customFrameOptionsValue: "SAMEORIGIN"
```

## UDP Ingress for VPN Traffic

VPN traffic (OpenVPN/WireGuard) uses UDP and requires special handling since standard Kubernetes Ingress only supports HTTP/HTTPS.

### Option 1: NodePort Service

Expose VPN ports directly via NodePort:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: gateway-vpn
  namespace: gatekey
spec:
  type: NodePort
  selector:
    app: gatekey-gateway
  ports:
    - name: openvpn
      protocol: UDP
      port: 1194
      targetPort: 1194
      nodePort: 31194
    - name: wireguard
      protocol: UDP
      port: 51820
      targetPort: 51820
      nodePort: 31820
```

### Option 2: LoadBalancer Service

For cloud environments with UDP LoadBalancer support:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: gateway-vpn
  namespace: gatekey
  annotations:
    # AWS NLB
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    # GCP
    cloud.google.com/l4-rbs: "enabled"
spec:
  type: LoadBalancer
  selector:
    app: gatekey-gateway
  ports:
    - name: openvpn
      protocol: UDP
      port: 1194
      targetPort: 1194
    - name: wireguard
      protocol: UDP
      port: 51820
      targetPort: 51820
```

### Option 3: MetalLB (Bare Metal)

For bare-metal clusters using MetalLB:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: gateway-vpn
  namespace: gatekey
  annotations:
    metallb.universe.tf/address-pool: vpn-pool
spec:
  type: LoadBalancer
  loadBalancerIP: 192.168.1.100  # Static IP from pool
  selector:
    app: gatekey-gateway
  ports:
    - name: openvpn
      protocol: UDP
      port: 1194
    - name: wireguard
      protocol: UDP
      port: 51820
```

### Option 4: Host Network

For single-node or edge deployments, use host networking:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gatekey-gateway
spec:
  template:
    spec:
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      containers:
        - name: gateway
          ports:
            - containerPort: 1194
              protocol: UDP
              hostPort: 1194
            - containerPort: 51820
              protocol: UDP
              hostPort: 51820
```

## Certificate Management

### cert-manager with Let's Encrypt

Automate TLS certificate management:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@yourcompany.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            class: nginx
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: gatekey-tls
  namespace: gatekey
spec:
  secretName: gatekey-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - vpn.yourcompany.com
```

### Using Existing Certificates

Create a TLS secret from existing certificates:

```bash
kubectl create secret tls gatekey-tls \
  --cert=fullchain.pem \
  --key=privkey.pem \
  -n gatekey
```

## External Load Balancers

### AWS Application Load Balancer

Use the AWS Load Balancer Controller:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gatekey
  namespace: gatekey
  annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTPS":443}]'
    alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:region:account:certificate/xxx
    alb.ingress.kubernetes.io/ssl-policy: ELBSecurityPolicy-TLS13-1-2-2021-06
spec:
  rules:
    - host: vpn.yourcompany.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: gatekey-web
                port:
                  number: 80
```

### Google Cloud Load Balancer

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gatekey
  namespace: gatekey
  annotations:
    kubernetes.io/ingress.class: "gce"
    kubernetes.io/ingress.global-static-ip-name: "gatekey-ip"
    networking.gke.io/managed-certificates: "gatekey-cert"
spec:
  rules:
    - host: vpn.yourcompany.com
      http:
        paths:
          - path: /*
            pathType: ImplementationSpecific
            backend:
              service:
                name: gatekey-web
                port:
                  number: 80
---
apiVersion: networking.gke.io/v1
kind: ManagedCertificate
metadata:
  name: gatekey-cert
  namespace: gatekey
spec:
  domains:
    - vpn.yourcompany.com
```

### Cloudflare Tunnel

For zero-trust access without exposing ports:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cloudflared
  namespace: gatekey
spec:
  replicas: 2
  selector:
    matchLabels:
      app: cloudflared
  template:
    metadata:
      labels:
        app: cloudflared
    spec:
      containers:
        - name: cloudflared
          image: cloudflare/cloudflared:latest
          args:
            - tunnel
            - --config
            - /etc/cloudflared/config.yaml
            - run
          volumeMounts:
            - name: config
              mountPath: /etc/cloudflared
              readOnly: true
            - name: creds
              mountPath: /etc/cloudflared/creds
              readOnly: true
      volumes:
        - name: config
          configMap:
            name: cloudflared-config
        - name: creds
          secret:
            secretName: cloudflared-creds
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cloudflared-config
  namespace: gatekey
data:
  config.yaml: |
    tunnel: your-tunnel-id
    credentials-file: /etc/cloudflared/creds/credentials.json
    ingress:
      - hostname: vpn.yourcompany.com
        service: http://gatekey-web:80
      - hostname: vpn.yourcompany.com
        path: /api/*
        service: http://gatekey-server:8080
      - service: http_status:404
```

## Health Checks

Configure appropriate health checks for your ingress:

### NGINX

```yaml
annotations:
  nginx.ingress.kubernetes.io/healthcheck-path: /health
  nginx.ingress.kubernetes.io/upstream-hash-by: "$remote_addr"
```

### Istio

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: gatekey-server
  namespace: gatekey
spec:
  host: gatekey-server
  trafficPolicy:
    connectionPool:
      http:
        h2UpgradePolicy: UPGRADE
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 30s
      baseEjectionTime: 30s
```

## Troubleshooting

### Check Ingress Status

```bash
# NGINX Ingress
kubectl get ingress -n gatekey
kubectl describe ingress gatekey -n gatekey

# Istio
kubectl get virtualservice -n gatekey
kubectl get gateway -n istio-system
istioctl analyze -n gatekey
```

### Test Connectivity

```bash
# Test HTTPS endpoint
curl -v https://vpn.yourcompany.com/health

# Test with specific DNS
curl -v --resolve vpn.yourcompany.com:443:INGRESS_IP https://vpn.yourcompany.com/health

# Test UDP (OpenVPN)
nc -vzu vpn.yourcompany.com 1194

# Test UDP (WireGuard)
nc -vzu vpn.yourcompany.com 51820
```

### View Ingress Controller Logs

```bash
# NGINX
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx

# Traefik
kubectl logs -n traefik -l app.kubernetes.io/name=traefik

# Istio
kubectl logs -n istio-system -l app=istio-ingressgateway
```

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| 502 Bad Gateway | Backend not ready | Check pod health, service endpoints |
| 503 Service Unavailable | No healthy backends | Verify deployment replicas, readiness probes |
| Certificate errors | TLS secret missing/invalid | Verify secret exists, check cert expiration |
| UDP traffic not working | Ingress doesn't support UDP | Use NodePort, LoadBalancer, or host network |
| Timeouts on large downloads | Proxy timeouts too low | Increase `proxy-read-timeout` annotation |
