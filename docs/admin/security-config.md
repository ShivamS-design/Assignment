# Security Configuration Guide

## Authentication Setup

### JWT Configuration
```yaml
# config/security.yml
auth:
  jwt:
    secret: "your-256-bit-secret-key-here"
    expiry: "24h"
    refresh_expiry: "168h"  # 7 days
    issuer: "wasm-as-os"
    audience: "wasm-users"
    
  providers:
    - type: "local"
      enabled: true
      config:
        password_policy:
          min_length: 8
          require_uppercase: true
          require_lowercase: true
          require_numbers: true
          require_symbols: true
          
    - type: "oauth2"
      enabled: true
      config:
        google:
          client_id: "your-google-client-id"
          client_secret: "your-google-client-secret"
          redirect_url: "https://yourdomain.com/auth/callback"
          
    - type: "ldap"
      enabled: false
      config:
        server: "ldap://ldap.company.com:389"
        bind_dn: "cn=admin,dc=company,dc=com"
        bind_password: "admin-password"
        user_base: "ou=users,dc=company,dc=com"
        user_filter: "(uid=%s)"
```

### Role-Based Access Control (RBAC)
```yaml
# config/rbac.yml
roles:
  admin:
    permissions:
      - "modules:*"
      - "users:*"
      - "system:*"
      - "scheduler:*"
      - "metrics:*"
      - "snapshots:*"
      
  developer:
    permissions:
      - "modules:read"
      - "modules:create"
      - "modules:execute"
      - "scheduler:read"
      - "scheduler:create"
      - "metrics:read"
      - "snapshots:read"
      
  viewer:
    permissions:
      - "modules:read"
      - "metrics:read"
      - "scheduler:read"
      
  operator:
    permissions:
      - "modules:read"
      - "modules:execute"
      - "scheduler:*"
      - "metrics:read"
      - "snapshots:create"
      - "snapshots:restore"

# Resource-based permissions
resources:
  modules:
    ownership: true  # Users can only access their own modules
    sharing: true    # Modules can be shared with other users
    
  tasks:
    ownership: true
    sharing: false
    
  snapshots:
    ownership: true
    sharing: true
```

### User Management
```bash
# Create admin user
./wasm-as-os admin create-user \
  --username admin \
  --password "SecurePassword123!" \
  --email admin@company.com \
  --roles admin

# Create developer user
./wasm-as-os admin create-user \
  --username developer1 \
  --password "DevPassword123!" \
  --email dev1@company.com \
  --roles developer

# Update user roles
./wasm-as-os admin update-user \
  --username developer1 \
  --add-roles operator

# Disable user
./wasm-as-os admin disable-user --username developer1

# List users
./wasm-as-os admin list-users --format table
```

## Network Security

### TLS/SSL Configuration
```yaml
# config/tls.yml
tls:
  enabled: true
  cert_file: "/etc/ssl/certs/wasm-as-os.crt"
  key_file: "/etc/ssl/private/wasm-as-os.key"
  min_version: "1.2"
  cipher_suites:
    - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
    - "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305"
    - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
  
  # Client certificate authentication (optional)
  client_auth:
    enabled: false
    ca_file: "/etc/ssl/ca/client-ca.crt"
    verify_mode: "require_and_verify"
```

### Firewall Rules
```bash
# UFW configuration
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (change port as needed)
sudo ufw allow 22/tcp

# Allow HTTPS
sudo ufw allow 443/tcp

# Allow HTTP (redirect to HTTPS)
sudo ufw allow 80/tcp

# Allow specific application ports
sudo ufw allow from 10.0.0.0/8 to any port 8080  # Internal API
sudo ufw allow from 10.0.0.0/8 to any port 5432  # PostgreSQL
sudo ufw allow from 10.0.0.0/8 to any port 6379  # Redis

# Enable firewall
sudo ufw enable
```

### Rate Limiting
```yaml
# config/rate_limit.yml
rate_limiting:
  enabled: true
  
  global:
    requests_per_minute: 1000
    burst: 100
    
  per_user:
    requests_per_minute: 100
    burst: 20
    
  per_endpoint:
    "/api/v1/auth/login":
      requests_per_minute: 5
      burst: 2
      window: "1m"
      
    "/api/v1/modules":
      requests_per_minute: 50
      burst: 10
      
    "/api/v1/modules/*/execute":
      requests_per_minute: 200
      burst: 50
      
  # IP-based blocking
  ip_whitelist:
    - "10.0.0.0/8"
    - "192.168.0.0/16"
    
  ip_blacklist:
    - "192.0.2.0/24"  # Example malicious network
```

## WASM Sandbox Security

### Capability Configuration
```yaml
# config/sandbox.yml
sandbox:
  default_capabilities:
    file_access: false
    network_access: false
    system_calls: false
    environment_access: false
    
  resource_limits:
    max_memory: "100MB"
    max_execution_time: "30s"
    max_stack_depth: 1000
    max_table_size: 1000
    max_globals: 100
    
  # Per-module capability overrides
  module_capabilities:
    "trusted-modules":
      file_access: true
      allowed_paths:
        - "/tmp/wasm-data"
        - "/var/lib/wasm-storage"
      network_access: true
      allowed_hosts:
        - "api.company.com"
        - "data.company.com"
        
  # Security policies
  policies:
    - name: "no_infinite_loops"
      enabled: true
      max_iterations: 1000000
      
    - name: "memory_bounds_check"
      enabled: true
      
    - name: "stack_overflow_protection"
      enabled: true
```

### Static Analysis Rules
```yaml
# config/static_analysis.yml
static_analysis:
  enabled: true
  
  rules:
    - name: "detect_infinite_loops"
      severity: "high"
      enabled: true
      
    - name: "detect_memory_leaks"
      severity: "medium"
      enabled: true
      
    - name: "detect_unsafe_operations"
      severity: "high"
      enabled: true
      
    - name: "check_import_restrictions"
      severity: "medium"
      enabled: true
      config:
        allowed_imports:
          - "env.memory"
          - "env.table"
        blocked_imports:
          - "env.exit"
          - "env.abort"
          
  # Quarantine policy
  quarantine:
    enabled: true
    high_risk_threshold: 8
    medium_risk_threshold: 5
    auto_quarantine: true
```

## Database Security

### PostgreSQL Hardening
```sql
-- Create dedicated database user
CREATE USER wasmuser WITH PASSWORD 'secure_random_password';
CREATE DATABASE wasmdb OWNER wasmuser;

-- Grant minimal permissions
GRANT CONNECT ON DATABASE wasmdb TO wasmuser;
GRANT USAGE ON SCHEMA public TO wasmuser;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO wasmuser;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO wasmuser;

-- Enable row-level security
ALTER TABLE modules ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- Create security policies
CREATE POLICY module_access_policy ON modules
  FOR ALL TO wasmuser
  USING (owner_id = current_setting('app.user_id')::uuid);

CREATE POLICY task_access_policy ON tasks
  FOR ALL TO wasmuser
  USING (user_id = current_setting('app.user_id')::uuid);
```

### Connection Security
```yaml
# config/database.yml
database:
  host: "localhost"
  port: 5432
  name: "wasmdb"
  user: "wasmuser"
  password: "${DB_PASSWORD}"  # Use environment variable
  
  # Connection pool settings
  max_connections: 25
  max_idle_connections: 5
  connection_lifetime: "1h"
  
  # SSL configuration
  ssl_mode: "require"
  ssl_cert: "/etc/ssl/certs/postgresql-client.crt"
  ssl_key: "/etc/ssl/private/postgresql-client.key"
  ssl_ca: "/etc/ssl/ca/postgresql-ca.crt"
  
  # Security settings
  statement_timeout: "30s"
  idle_in_transaction_timeout: "60s"
  log_statement: "all"  # For audit purposes
```

## Audit Logging

### Audit Configuration
```yaml
# config/audit.yml
audit:
  enabled: true
  
  # Log destinations
  destinations:
    - type: "file"
      path: "/var/log/wasm-as-os/audit.log"
      rotation:
        max_size: "100MB"
        max_files: 10
        compress: true
        
    - type: "syslog"
      facility: "local0"
      tag: "wasm-as-os"
      
    - type: "database"
      table: "audit_logs"
      
  # Events to audit
  events:
    authentication:
      - "login_success"
      - "login_failure"
      - "logout"
      - "token_refresh"
      
    authorization:
      - "permission_denied"
      - "role_change"
      
    modules:
      - "module_upload"
      - "module_delete"
      - "module_execute"
      
    system:
      - "config_change"
      - "user_create"
      - "user_delete"
      - "service_start"
      - "service_stop"
      
  # Sensitive data filtering
  filter_sensitive_data: true
  sensitive_fields:
    - "password"
    - "token"
    - "secret"
    - "key"
```

### Log Analysis
```bash
# Search for failed login attempts
grep "login_failure" /var/log/wasm-as-os/audit.log | tail -20

# Monitor privilege escalation attempts
grep "permission_denied" /var/log/wasm-as-os/audit.log | grep "admin"

# Check module execution patterns
grep "module_execute" /var/log/wasm-as-os/audit.log | \
  jq '.module_id' | sort | uniq -c | sort -nr

# Generate security report
./wasm-as-os admin security-report \
  --start-date "2024-01-01" \
  --end-date "2024-01-31" \
  --format pdf \
  --output security-report-jan-2024.pdf
```

## Backup and Recovery

### Backup Strategy
```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/var/backups/wasm-as-os"
DATE=$(date +%Y%m%d_%H%M%S)

# Database backup
pg_dump -h localhost -U wasmuser wasmdb | \
  gzip > "$BACKUP_DIR/db_backup_$DATE.sql.gz"

# Configuration backup
tar -czf "$BACKUP_DIR/config_backup_$DATE.tar.gz" \
  /etc/wasm-as-os/

# Module storage backup
tar -czf "$BACKUP_DIR/modules_backup_$DATE.tar.gz" \
  /var/lib/wasm-as-os/modules/

# Cleanup old backups (keep 30 days)
find "$BACKUP_DIR" -name "*.gz" -mtime +30 -delete

# Upload to secure storage
aws s3 cp "$BACKUP_DIR/" s3://company-backups/wasm-as-os/ \
  --recursive --exclude "*" --include "*$DATE*"
```

### Disaster Recovery
```bash
#!/bin/bash
# restore.sh

BACKUP_DATE="$1"
BACKUP_DIR="/var/backups/wasm-as-os"

if [ -z "$BACKUP_DATE" ]; then
  echo "Usage: $0 <backup_date>"
  exit 1
fi

# Stop services
systemctl stop wasm-as-os

# Restore database
gunzip -c "$BACKUP_DIR/db_backup_$BACKUP_DATE.sql.gz" | \
  psql -h localhost -U wasmuser wasmdb

# Restore configuration
tar -xzf "$BACKUP_DIR/config_backup_$BACKUP_DATE.tar.gz" -C /

# Restore modules
tar -xzf "$BACKUP_DIR/modules_backup_$BACKUP_DATE.tar.gz" -C /

# Start services
systemctl start wasm-as-os

echo "Restore completed for backup date: $BACKUP_DATE"
```

## Security Monitoring

### Intrusion Detection
```yaml
# config/ids.yml
intrusion_detection:
  enabled: true
  
  rules:
    - name: "brute_force_login"
      condition: "failed_logins > 5 in 5m"
      action: "block_ip"
      duration: "1h"
      
    - name: "suspicious_module_upload"
      condition: "module_size > 50MB or contains_suspicious_patterns"
      action: "quarantine"
      
    - name: "privilege_escalation"
      condition: "permission_denied and target_resource = admin"
      action: "alert"
      
    - name: "unusual_execution_pattern"
      condition: "execution_rate > 1000/min"
      action: "throttle"
      
  # Alert destinations
  alerts:
    - type: "email"
      recipients: ["security@company.com"]
      
    - type: "slack"
      webhook: "https://hooks.slack.com/services/..."
      
    - type: "syslog"
      facility: "security"
```

### Security Metrics
```bash
# Monitor security metrics
curl -s http://localhost:8080/metrics | grep security

# Key security metrics to monitor:
# - authentication_failures_total
# - authorization_denials_total
# - suspicious_modules_detected_total
# - rate_limit_violations_total
# - sandbox_violations_total
```

## Compliance

### GDPR Compliance
```yaml
# config/privacy.yml
privacy:
  gdpr_compliance: true
  
  data_retention:
    audit_logs: "7y"      # 7 years for audit
    user_data: "2y"       # 2 years after last login
    module_data: "1y"     # 1 year after last use
    metrics_data: "90d"   # 90 days for metrics
    
  data_anonymization:
    enabled: true
    fields_to_anonymize:
      - "ip_address"
      - "user_agent"
      - "email"  # Keep hash for uniqueness
      
  right_to_be_forgotten:
    enabled: true
    automated_deletion: true
    
  data_export:
    enabled: true
    formats: ["json", "csv"]
    encryption: true
```

### SOC 2 Compliance
```yaml
# config/soc2.yml
soc2:
  security:
    access_controls: true
    multi_factor_auth: true
    encryption_at_rest: true
    encryption_in_transit: true
    
  availability:
    monitoring: true
    alerting: true
    backup_testing: true
    disaster_recovery: true
    
  processing_integrity:
    input_validation: true
    error_handling: true
    data_integrity_checks: true
    
  confidentiality:
    data_classification: true
    access_logging: true
    secure_disposal: true
    
  privacy:
    consent_management: true
    data_minimization: true
    purpose_limitation: true
```