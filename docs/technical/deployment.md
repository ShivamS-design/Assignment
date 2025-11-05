# Deployment Guide

## Quick Deployment

### Docker Compose (Recommended)
```bash
# Clone repository
git clone https://github.com/company/wasm-as-os.git
cd wasm-as-os

# Start all services
docker-compose up -d

# Check status
docker-compose ps
```

### Single Command Setup
```bash
# Development environment
make dev-setup

# Production environment
make prod-deploy
```

## Production Deployment

### Prerequisites
- Docker 20.10+
- Docker Compose 2.0+
- 4GB RAM minimum
- 20GB disk space

### Environment Configuration
```bash
# Copy environment template
cp .env.example .env

# Edit configuration
vim .env
```

**Required Environment Variables:**
```env
# Database
POSTGRES_HOST=postgres
POSTGRES_DB=wasmdb
POSTGRES_USER=wasmuser
POSTGRES_PASSWORD=secure_password

# Redis
REDIS_HOST=redis
REDIS_PASSWORD=redis_password

# JWT
JWT_SECRET=your-256-bit-secret
JWT_EXPIRY=24h

# Security
RATE_LIMIT=1000
CORS_ORIGINS=https://yourdomain.com

# Resources
MAX_MEMORY=1GB
MAX_EXECUTION_TIME=30s
```

### SSL/TLS Setup
```bash
# Generate certificates
mkdir -p certs
openssl req -x509 -newkey rsa:4096 -keyout certs/key.pem -out certs/cert.pem -days 365 -nodes

# Update docker-compose.yml
services:
  api:
    volumes:
      - ./certs:/app/certs
    environment:
      - TLS_CERT_FILE=/app/certs/cert.pem
      - TLS_KEY_FILE=/app/certs/key.pem
```

### Load Balancer Configuration

**Nginx Configuration:**
```nginx
upstream wasm_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}

server {
    listen 443 ssl http2;
    server_name wasm.yourdomain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location /api/ {
        proxy_pass http://wasm_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    location /ws/ {
        proxy_pass http://wasm_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    
    location / {
        root /var/www/wasm-frontend;
        try_files $uri $uri/ /index.html;
    }
}
```

## Kubernetes Deployment

### Namespace Setup
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: wasm-as-os
```

### ConfigMap
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: wasm-config
  namespace: wasm-as-os
data:
  config.yaml: |
    server:
      port: 8080
      host: 0.0.0.0
    database:
      host: postgres-service
      port: 5432
      name: wasmdb
    redis:
      host: redis-service
      port: 6379
```

### Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wasm-api
  namespace: wasm-as-os
spec:
  replicas: 3
  selector:
    matchLabels:
      app: wasm-api
  template:
    metadata:
      labels:
        app: wasm-api
    spec:
      containers:
      - name: api
        image: wasm-as-os:latest
        ports:
        - containerPort: 8080
        env:
        - name: CONFIG_FILE
          value: /etc/config/config.yaml
        volumeMounts:
        - name: config
          mountPath: /etc/config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: config
        configMap:
          name: wasm-config
```

### Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: wasm-api-service
  namespace: wasm-as-os
spec:
  selector:
    app: wasm-api
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## Database Migration

### Initial Setup
```bash
# Run migrations
docker-compose exec api go run cmd/migrate/main.go up

# Create admin user
docker-compose exec api go run cmd/admin/main.go create-user \
  --username admin \
  --password admin123 \
  --roles admin
```

### Backup and Restore
```bash
# Backup
docker-compose exec postgres pg_dump -U wasmuser wasmdb > backup.sql

# Restore
docker-compose exec -T postgres psql -U wasmuser wasmdb < backup.sql
```

## Monitoring Setup

### Prometheus Configuration
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'wasm-as-os'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

### Grafana Dashboard
```json
{
  "dashboard": {
    "title": "WASM-as-OS Metrics",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ]
      }
    ]
  }
}
```

## Health Checks

### Application Health
```bash
# API health
curl http://localhost:8080/health

# Database connectivity
curl http://localhost:8080/health/db

# Redis connectivity
curl http://localhost:8080/health/redis
```

### Kubernetes Probes
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Scaling Configuration

### Horizontal Pod Autoscaler
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: wasm-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: wasm-api
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Troubleshooting

### Common Issues

**Connection Refused:**
```bash
# Check service status
docker-compose ps
kubectl get pods -n wasm-as-os

# Check logs
docker-compose logs api
kubectl logs -f deployment/wasm-api -n wasm-as-os
```

**Database Connection:**
```bash
# Test connection
docker-compose exec postgres psql -U wasmuser -d wasmdb -c "SELECT 1;"

# Check configuration
docker-compose exec api env | grep POSTGRES
```

**Memory Issues:**
```bash
# Check resource usage
docker stats
kubectl top pods -n wasm-as-os

# Adjust limits
vim docker-compose.yml  # or k8s manifests
```