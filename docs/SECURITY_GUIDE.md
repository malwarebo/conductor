# Conductor Security Best Practices Guide

## Overview

This guide outlines security best practices for deploying and operating Conductor in production environments. It covers authentication, data protection, network security, and compliance considerations.

## Table of Contents

1. [Authentication and Authorization](#authentication-and-authorization)
2. [Data Protection](#data-protection)
3. [Network Security](#network-security)
4. [API Security](#api-security)
5. [Database Security](#database-security)
6. [Infrastructure Security](#infrastructure-security)
7. [Monitoring and Incident Response](#monitoring-and-incident-response)
8. [Compliance and Auditing](#compliance-and-auditing)
9. [Security Checklist](#security-checklist)

## Authentication and Authorization

### JWT Token Security

#### Token Generation
```go
// Use strong, random secrets (minimum 32 characters)
JWT_SECRET=$(openssl rand -base64 32)

// Set appropriate expiration times
const (
    AccessTokenExpiry  = 15 * time.Minute
    RefreshTokenExpiry = 7 * 24 * time.Hour
)
```

#### Token Validation
```go
// Always validate token signature and expiration
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    // Verify signature
    if !j.verifySignature(tokenString) {
        return nil, errors.New("invalid signature")
    }
    
    // Check expiration
    if claims.ExpiresAt < time.Now().Unix() {
        return nil, errors.New("token expired")
    }
    
    return claims, nil
}
```

#### Best Practices
- Use short-lived access tokens (15 minutes)
- Implement refresh token rotation
- Store tokens securely (httpOnly cookies preferred)
- Implement token blacklisting for logout

### API Key Management

#### Key Generation
```go
// Generate cryptographically secure API keys
func (m *APIKeyManager) GenerateKey(name, userID string, scopes []string) (*APIKey, string, error) {
    keyBytes := make([]byte, 32)
    if _, err := rand.Read(keyBytes); err != nil {
        return nil, "", err
    }
    
    key := fmt.Sprintf("conductor_%s", hex.EncodeToString(keyBytes))
    keyHash := m.hashKey(key)
    
    return &APIKey{
        ID:        generateID(),
        KeyHash:   keyHash,
        Scopes:    scopes,
        UserID:    userID,
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(365 * 24 * time.Hour), // 1 year
        IsActive:  true,
    }, key, nil
}
```

#### Key Rotation
```go
// Implement automatic key rotation
func (m *APIKeyManager) RotateKey(keyID string) (*APIKey, string, error) {
    // Deactivate old key
    if err := m.RevokeKey(keyID); err != nil {
        return nil, "", err
    }
    
    // Generate new key
    return m.GenerateKey("rotated_key", userID, scopes)
}
```

### Role-Based Access Control (RBAC)

#### Role Definition
```go
type Role struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Permissions []string `json:"permissions"`
    CreatedAt   time.Time `json:"created_at"`
}

// Define roles with specific permissions
var Roles = map[string][]string{
    "admin": {
        "payments:create",
        "payments:read",
        "payments:update",
        "payments:delete",
        "subscriptions:create",
        "subscriptions:read",
        "subscriptions:update",
        "subscriptions:delete",
        "disputes:read",
        "disputes:update",
        "analytics:read",
        "system:admin",
    },
    "merchant": {
        "payments:create",
        "payments:read",
        "subscriptions:create",
        "subscriptions:read",
        "analytics:read",
    },
    "viewer": {
        "payments:read",
        "subscriptions:read",
        "analytics:read",
    },
}
```

#### Permission Checking
```go
func (m *AuthMiddleware) CheckPermission(userRoles []string, requiredPermission string) bool {
    for _, role := range userRoles {
        permissions, exists := Roles[role]
        if !exists {
            continue
        }
        
        for _, permission := range permissions {
            if permission == requiredPermission {
                return true
            }
        }
    }
    return false
}
```

## Data Protection

### Encryption at Rest

#### Database Encryption
```sql
-- Enable transparent data encryption for sensitive tables
CREATE TABLE payments_encrypted (
    id VARCHAR(255) PRIMARY KEY,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    encrypted_data BYTEA, -- Encrypted sensitive data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Application-Level Encryption
```go
// Encrypt sensitive data before storing
func (e *EncryptionManager) EncryptSensitiveData(data string) (string, error) {
    // Use AES-256-GCM for authenticated encryption
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

### Encryption in Transit

#### TLS Configuration
```go
// Configure TLS with strong ciphers
tlsConfig := &tls.Config{
    MinVersion:               tls.VersionTLS12,
    CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
    PreferServerCipherSuites: true,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
    },
}
```

#### Database SSL
```bash
# Configure PostgreSQL SSL
export DB_SSL_MODE=require
export DB_SSL_CERT=/path/to/client-cert.pem
export DB_SSL_KEY=/path/to/client-key.pem
export DB_SSL_ROOT_CERT=/path/to/ca-cert.pem
```

### Data Masking and Anonymization

#### PII Masking
```go
func MaskPII(data string) string {
    if len(data) < 4 {
        return "****"
    }
    
    // Show first 2 and last 2 characters
    return data[:2] + "****" + data[len(data)-2:]
}

// Example: "john.doe@example.com" -> "jo****om"
```

#### Audit Log Anonymization
```go
func (a *AuditLogger) LogPaymentEvent(ctx context.Context, event string, payment *models.Payment) {
    // Anonymize sensitive data in logs
    anonymizedPayment := &models.Payment{
        ID:       payment.ID,
        Amount:   payment.Amount,
        Currency: payment.Currency,
        Status:   payment.Status,
        // Don't log sensitive fields
    }
    
    a.logger.Info("Payment event", map[string]interface{}{
        "event":   event,
        "payment": anonymizedPayment,
        "user_id": ctx.Value("user_id"),
    })
}
```

## Network Security

### Firewall Configuration

#### UFW Rules
```bash
# Allow only necessary ports
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (change port for security)
sudo ufw allow 2222/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow database connections (internal only)
sudo ufw allow from 10.0.0.0/8 to any port 5432
sudo ufw allow from 172.16.0.0/12 to any port 5432
sudo ufw allow from 192.168.0.0/16 to any port 5432

# Allow Redis connections (internal only)
sudo ufw allow from 10.0.0.0/8 to any port 6379
sudo ufw allow from 172.16.0.0/12 to any port 6379
sudo ufw allow from 192.168.0.0/16 to any port 6379

# Enable firewall
sudo ufw enable
```

#### iptables Rules
```bash
# Create custom iptables rules
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 5432 -s 10.0.0.0/8 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 5432 -s 172.16.0.0/12 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 5432 -s 192.168.0.0/16 -j ACCEPT
sudo iptables -A INPUT -j DROP
```

### VPN and Private Networks

#### WireGuard Configuration
```ini
# /etc/wireguard/wg0.conf
[Interface]
PrivateKey = <server_private_key>
Address = 10.0.0.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

[Peer]
PublicKey = <client_public_key>
AllowedIPs = 10.0.0.2/32
```

### DDoS Protection

#### Rate Limiting
```go
// Implement tiered rate limiting
type TieredRateLimiter struct {
    tiers map[string]*rate.Limiter
}

func (trl *TieredRateLimiter) GetLimiter(tier string) *rate.Limiter {
    switch tier {
    case "premium":
        return rate.NewLimiter(rate.Limit(1000), 2000) // 1000 RPS, burst 2000
    case "standard":
        return rate.NewLimiter(rate.Limit(100), 200)   // 100 RPS, burst 200
    case "basic":
        return rate.NewLimiter(rate.Limit(10), 20)     // 10 RPS, burst 20
    default:
        return rate.NewLimiter(rate.Limit(1), 2)       // 1 RPS, burst 2
    }
}
```

#### IP Blocking
```go
// Implement IP blocking for malicious requests
type IPBlocker struct {
    blockedIPs map[string]time.Time
    mu          sync.RWMutex
}

func (ib *IPBlocker) BlockIP(ip string, duration time.Duration) {
    ib.mu.Lock()
    defer ib.mu.Unlock()
    
    ib.blockedIPs[ip] = time.Now().Add(duration)
}

func (ib *IPBlocker) IsBlocked(ip string) bool {
    ib.mu.RLock()
    defer ib.mu.RUnlock()
    
    blockTime, exists := ib.blockedIPs[ip]
    if !exists {
        return false
    }
    
    if time.Now().After(blockTime) {
        delete(ib.blockedIPs, ip)
        return false
    }
    
    return true
}
```

## API Security

### Input Validation

#### Request Validation
```go
// Validate all incoming requests
func (v *Validator) ValidatePaymentRequest(req *CreatePaymentRequest) error {
    if req.Amount <= 0 {
        return errors.New("amount must be positive")
    }
    
    if req.Amount > 1000000 { // $10,000 limit
        return errors.New("amount exceeds maximum limit")
    }
    
    if !isValidCurrency(req.Currency) {
        return errors.New("invalid currency code")
    }
    
    if !isValidEmail(req.CustomerEmail) {
        return errors.New("invalid email format")
    }
    
    return nil
}
```

#### SQL Injection Prevention
```go
// Use parameterized queries
func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*models.Payment, error) {
    var payment models.Payment
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&payment).Error
    if err != nil {
        return nil, err
    }
    return &payment, nil
}
```

### CORS Configuration

#### Secure CORS Setup
```go
func CreateCORSMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            
            // Allow only trusted origins
            allowedOrigins := []string{
                "https://yourdomain.com",
                "https://app.yourdomain.com",
            }
            
            if isAllowedOrigin(origin, allowedOrigins) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
            }
            
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
            w.Header().Set("Access-Control-Allow-Credentials", "true")
            w.Header().Set("Access-Control-Max-Age", "86400")
            
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### Security Headers

#### HTTP Security Headers
```go
func (am *AuthMiddleware) HeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Prevent XSS attacks
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // Enforce HTTPS
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
        
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
        
        // Referrer Policy
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        // Permissions Policy
        w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        
        next.ServeHTTP(w, r)
    })
}
```

## Database Security

### Connection Security

#### SSL/TLS Configuration
```go
// Configure database connection with SSL
func createDBConnection(cfg *config.Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
        cfg.Database.Host,
        cfg.Database.User,
        cfg.Database.Password,
        cfg.Database.Name,
        cfg.Database.Port,
        cfg.Database.SSLMode,
    )
    
    if cfg.Database.SSLMode == "require" {
        dsn += " sslcert=" + cfg.Database.SSLCert
        dsn += " sslkey=" + cfg.Database.SSLKey
        dsn += " sslrootcert=" + cfg.Database.SSLRootCert
    }
    
    return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
```

#### Connection Pooling Security
```go
// Configure secure connection pooling
func (cp *ConnectionPool) ConfigurePool() {
    cp.db.SetMaxOpenConns(100)
    cp.db.SetMaxIdleConns(10)
    cp.db.SetConnMaxLifetime(time.Hour)
    cp.db.SetConnMaxIdleTime(time.Minute * 30)
}
```

### Database Access Control

#### User Permissions
```sql
-- Create application user with minimal permissions
CREATE USER conductor_app WITH PASSWORD 'secure_password';

-- Grant only necessary permissions
GRANT CONNECT ON DATABASE conductor_prod TO conductor_app;
GRANT USAGE ON SCHEMA public TO conductor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO conductor_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO conductor_app;

-- Revoke unnecessary permissions
REVOKE CREATE ON DATABASE conductor_prod FROM conductor_app;
REVOKE DROP ON DATABASE conductor_prod FROM conductor_app;
```

#### Row-Level Security
```sql
-- Enable row-level security
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;

-- Create policy for user-specific access
CREATE POLICY payment_user_policy ON payments
    FOR ALL TO conductor_app
    USING (user_id = current_setting('app.current_user_id'));
```

### Database Encryption

#### Transparent Data Encryption
```sql
-- Create encrypted tables
CREATE TABLE payments_encrypted (
    id VARCHAR(255) PRIMARY KEY,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    encrypted_data BYTEA,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create encryption function
CREATE OR REPLACE FUNCTION encrypt_payment_data(data TEXT)
RETURNS BYTEA AS $$
BEGIN
    RETURN pgp_sym_encrypt(data, current_setting('app.encryption_key'));
END;
$$ LANGUAGE plpgsql;
```

## Infrastructure Security

### Container Security

#### Docker Security
```dockerfile
# Use minimal base image
FROM golang:1.21-alpine AS builder

# Create non-root user
RUN adduser -D -s /bin/sh conductor

# Copy application
COPY --chown=conductor:conductor . /app
WORKDIR /app

# Build application
RUN go build -o conductor main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy binary
COPY --from=builder /app/conductor .

# Run as non-root user
USER conductor

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Start application
CMD ["./conductor"]
```

#### Kubernetes Security
```yaml
# Security context
apiVersion: apps/v1
kind: Deployment
metadata:
  name: conductor
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: conductor
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
          capabilities:
            drop:
            - ALL
```

### Secrets Management

#### Kubernetes Secrets
```yaml
# Create secrets
apiVersion: v1
kind: Secret
metadata:
  name: conductor-secrets
type: Opaque
data:
  jwt-secret: <base64-encoded-secret>
  db-password: <base64-encoded-password>
  encryption-key: <base64-encoded-key>
```

#### Environment Variable Security
```bash
# Use environment variables for secrets
export JWT_SECRET=$(openssl rand -base64 32)
export DB_PASSWORD=$(openssl rand -base64 32)
export ENCRYPTION_KEY=$(openssl rand -base64 32)

# Never commit secrets to version control
echo "*.env" >> .gitignore
echo "secrets/" >> .gitignore
```

### Network Segmentation

#### VPC Configuration
```yaml
# AWS VPC example
Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16
      EnableDnsHostnames: true
      EnableDnsSupport: true
      Tags:
        - Key: Name
          Value: Conductor-VPC

  PrivateSubnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      CidrBlock: 10.0.1.0/24
      AvailabilityZone: us-west-2a
      Tags:
        - Key: Name
          Value: Conductor-Private-Subnet
```

## Monitoring and Incident Response

### Security Monitoring

#### Log Analysis
```go
// Monitor security events
func (m *SecurityMonitor) LogSecurityEvent(event string, details map[string]interface{}) {
    m.logger.Warn("Security event", map[string]interface{}{
        "event":   event,
        "details": details,
        "timestamp": time.Now(),
        "ip":     details["ip"],
        "user_id": details["user_id"],
    })
    
    // Send alert for critical events
    if m.isCriticalEvent(event) {
        m.sendAlert(event, details)
    }
}
```

#### Intrusion Detection
```go
// Detect suspicious activity
func (m *SecurityMonitor) DetectIntrusion(ip string, userID string) bool {
    // Check for multiple failed login attempts
    failedAttempts := m.getFailedAttempts(ip, userID)
    if failedAttempts > 5 {
        m.LogSecurityEvent("multiple_failed_logins", map[string]interface{}{
            "ip": ip,
            "user_id": userID,
            "attempts": failedAttempts,
        })
        return true
    }
    
    // Check for unusual request patterns
    if m.isUnusualPattern(ip, userID) {
        m.LogSecurityEvent("unusual_pattern", map[string]interface{}{
            "ip": ip,
            "user_id": userID,
        })
        return true
    }
    
    return false
}
```

### Incident Response

#### Automated Response
```go
// Automatically respond to security incidents
func (m *SecurityMonitor) HandleSecurityIncident(event string, details map[string]interface{}) {
    switch event {
    case "multiple_failed_logins":
        // Block IP temporarily
        m.blockIP(details["ip"].(string), 15*time.Minute)
        
    case "suspicious_activity":
        // Require additional authentication
        m.requireMFA(details["user_id"].(string))
        
    case "data_breach":
        // Immediate response
        m.notifySecurityTeam(event, details)
        m.rotateSecrets()
    }
}
```

#### Alerting
```go
// Send security alerts
func (m *SecurityMonitor) sendAlert(event string, details map[string]interface{}) {
    alert := &SecurityAlert{
        Event:     event,
        Details:   details,
        Timestamp: time.Now(),
        Severity:  m.getSeverity(event),
    }
    
    // Send to multiple channels
    m.sendEmailAlert(alert)
    m.sendSlackAlert(alert)
    m.sendPagerDutyAlert(alert)
}
```

## Compliance and Auditing

### PCI DSS Compliance

#### Data Protection
```go
// Implement PCI DSS requirements
func (p *PCIManager) ProcessPaymentData(data *PaymentData) error {
    // Encrypt sensitive data
    encryptedData, err := p.encryptSensitiveData(data)
    if err != nil {
        return err
    }
    
    // Store encrypted data
    return p.storeEncryptedData(encryptedData)
}
```

#### Audit Logging
```go
// Comprehensive audit logging
func (a *AuditLogger) LogPaymentEvent(ctx context.Context, event string, payment *models.Payment) {
    auditLog := &AuditLog{
        Event:     event,
        UserID:    ctx.Value("user_id").(string),
        IP:        ctx.Value("ip").(string),
        UserAgent: ctx.Value("user_agent").(string),
        PaymentID: payment.ID,
        Amount:    payment.Amount,
        Currency:  payment.Currency,
        Timestamp: time.Now(),
    }
    
    a.logger.Info("Audit log", auditLog)
}
```

### GDPR Compliance

#### Data Subject Rights
```go
// Implement GDPR data subject rights
func (g *GDPRManager) HandleDataSubjectRequest(request *DataSubjectRequest) error {
    switch request.Type {
    case "access":
        return g.provideDataAccess(request)
    case "rectification":
        return g.rectifyData(request)
    case "erasure":
        return g.eraseData(request)
    case "portability":
        return g.exportData(request)
    }
    return nil
}
```

#### Data Retention
```go
// Implement data retention policies
func (r *RetentionManager) ApplyRetentionPolicy() error {
    // Delete old audit logs
    cutoffDate := time.Now().AddDate(-7, 0, 0) // 7 years
    return r.db.Where("created_at < ?", cutoffDate).Delete(&AuditLog{}).Error
}
```

