FROM nginx:alpine

# OCI labels for supply chain security
LABEL org.opencontainers.image.title="GateKey Web UI"
LABEL org.opencontainers.image.description="GateKey Zero-Trust VPN Web Interface"
LABEL org.opencontainers.image.vendor="Dye Tech"
LABEL org.opencontainers.image.source="https://github.com/dye-tech/GateKey"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Upgrade all packages to get security fixes, then remove default config
RUN apk upgrade --no-cache && \
    rm /etc/nginx/conf.d/default.conf

COPY deploy/nginx.conf /etc/nginx/conf.d/gatex.conf

# Copy built frontend
COPY web/dist /usr/share/nginx/html

# Create non-root user with explicit UID
RUN addgroup -g 65532 -S gatekey && \
    adduser -u 65532 -S -G gatekey -h /app -s /sbin/nologin gatekey && \
    chown -R 65532:65532 /usr/share/nginx/html && \
    chown -R 65532:65532 /var/cache/nginx && \
    chown -R 65532:65532 /var/log/nginx && \
    chown -R 65532:65532 /etc/nginx/conf.d && \
    touch /var/run/nginx.pid && \
    chown 65532:65532 /var/run/nginx.pid

# Switch to non-root user
USER 65532:65532

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
