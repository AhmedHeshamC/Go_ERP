# ERPGo Security Best Practices Guide

## Overview

This comprehensive security guide covers all aspects of securing the ERPGo system, including application security, infrastructure security, data protection, and compliance requirements. Following these practices helps ensure the confidentiality, integrity, and availability of the ERP system and its data.

## Table of Contents

1. [Security Architecture](#security-architecture)
2. [Authentication & Authorization](#authentication--authorization)
3. [Data Protection](#data-protection)
4. [Network Security](#network-security)
5. [Application Security](#application-security)
6. [Infrastructure Security](#infrastructure-security)
7. [Database Security](#database-security)
8. [API Security](#api-security)
9. [Logging & Monitoring](#logging--monitoring)
10. [Incident Response](#incident-response)
11. [Compliance & Auditing](#compliance--auditing)
12. [Security Checklist](#security-checklist)

## Security Architecture

### Defense in Depth Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                    Network Security Layer                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  │   WAF       │ │   Firewall  │ │   DDoS      │ │    VPN      │ │
│  │ Protection  │ │   Rules     │ │ Protection  │ │   Access    │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                   Application Security Layer                      │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  │   Input     │ │   Output    │ │  Session    │ │    CSRF     │ │
│  │ Validation  │ │  Encoding   │ │ Management  │ │ Protection  │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                   Authorization Layer                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  │     RBAC    │ │   JWT       │ │  Rate       │ │  Resource   │ │
│  │  System     │ │  Tokens     │ │ Limiting    │ │  Permissions│ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                      Data Protection Layer                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  │ Encryption  │ │  Hashing    │ │   Key       │ │    Access   │ │
│ │  at Rest    │ │  Functions  │ │ Management  │ │   Control   │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                    Infrastructure Security                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  │  Container  │ │   Network   │ │   Host      │ │   Backup    │ │
│  │ Security   │ │ Segmentation│ │ Hardening   │ │  Security   │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Security Principles

1. **Principle of Least Privilege**: Users and services only have access to resources they absolutely need
2. **Zero Trust Architecture**: Never trust, always verify
3. **Defense in Depth**: Multiple layers of security controls
4. **Security by Design**: Security built into every layer
5. **Fail Securely**: System fails to a secure state when errors occur

## Authentication & Authorization

### Secure Password Management

#### Password Policy Implementation

```go
// pkg/auth/password.go
package auth

import (
    "crypto/rand"
    "encoding/base64"
    "errors"
    "regexp"
    "unicode/utf8"

    "golang.org/x/crypto/bcrypt"
)

type PasswordPolicy struct {
    MinLength        int
    MaxLength        int
    RequireUppercase bool
    RequireLowercase bool
    RequireNumbers   bool
    RequireSymbols   bool
    ForbiddenPatterns []string
}

var DefaultPasswordPolicy = &PasswordPolicy{
    MinLength:        12,
    MaxLength:        128,
    RequireUppercase: true,
    RequireLowercase: true,
    RequireNumbers:   true,
    RequireSymbols:   true,
    ForbiddenPatterns: []string{"password", "123456", "qwerty"},
}

func ValidatePassword(password string, policy *PasswordPolicy) error {
    length := utf8.RuneCountInString(password)

    if length < policy.MinLength {
        return errors.New("password too short")
    }

    if length > policy.MaxLength {
        return errors.New("password too long")
    }

    if policy.RequireUppercase && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
        return errors.New("password must contain uppercase letter")
    }

    if policy.RequireLowercase && !regexp.MustCompile(`[a-z]`).MatchString(password) {
        return errors.New("password must contain lowercase letter")
    }

    if policy.RequireNumbers && !regexp.MustCompile(`\d`).MatchString(password) {
        return errors.New("password must contain number")
    }

    if policy.RequireSymbols && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password) {
        return errors.New("password must contain special character")
    }

    // Check forbidden patterns
    lowerPassword := strings.ToLower(password)
    for _, pattern := range policy.ForbiddenPatterns {
        if strings.Contains(lowerPassword, pattern) {
            return errors.New("password contains forbidden pattern")
        }
    }

    return nil
}

func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hash), nil
}

func VerifyPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

func GenerateSecureToken(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}
```

#### Multi-Factor Authentication (MFA)

```go
// pkg/auth/mfa.go
package auth

import (
    "crypto/hmac"
    "crypto/sha1"
    "encoding/base32"
    "encoding/binary"
    "fmt"
    "math"
    "time"

    "github.com/pquerna/otp/totp"
)

type MFAService struct {
    issuer string
}

func NewMFAService(issuer string) *MFAService {
    return &MFAService{issuer: issuer}
}

func (m *MFAService) GenerateSecret() (string, error) {
    return GenerateSecureToken(20)
}

func (m *MFAService) GenerateQRCode(secret, userEmail string) (string, error) {
    key, err := otp.NewKeyFromSecret(secret)
    if err != nil {
        return "", err
    }

    key.Issuer = m.issuer
    key.AccountName = userEmail

    return key.URL(), nil
}

func (m *MFAService) VerifyCode(secret, code string) bool {
    return totp.Validate(code, secret)
}

// Rate limiting for MFA attempts
type MFARateLimiter struct {
    attempts map[string][]time.Time
    maxAttempts int
    window time.Duration
    mutex sync.RWMutex
}

func NewMFARateLimiter(maxAttempts int, window time.Duration) *MFARateLimiter {
    return &MFARateLimiter{
        attempts: make(map[string][]time.Time),
        maxAttempts: maxAttempts,
        window: window,
    }
}

func (r *MFARateLimiter) CheckAttempts(userID string) bool {
    r.mutex.Lock()
    defer r.mutex.Unlock()

    now := time.Now()
    attempts := r.attempts[userID]

    // Remove old attempts outside the window
    var validAttempts []time.Time
    for _, attempt := range attempts {
        if now.Sub(attempt) < r.window {
            validAttempts = append(validAttempts, attempt)
        }
    }

    if len(validAttempts) >= r.maxAttempts {
        return false
    }

    // Add current attempt
    validAttempts = append(validAttempts, now)
    r.attempts[userID] = validAttempts

    return true
}
```

### JWT Token Security

#### Secure JWT Implementation

```go
// pkg/auth/jwt.go
package auth

import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
    secretKey     []byte
    refreshSecret []byte
    issuer        string
    tokenExpiry   time.Duration
    refreshExpiry time.Duration
}

type Claims struct {
    UserID   string   `json:"user_id"`
    Email    string   `json:"email"`
    Roles    []string `json:"roles"`
    jwt.RegisteredClaims
}

func NewJWTService(secretKey, refreshSecret, issuer string, tokenExpiry, refreshExpiry time.Duration) *JWTService {
    return &JWTService{
        secretKey:     []byte(secretKey),
        refreshSecret: []byte(refreshSecret),
        issuer:        issuer,
        tokenExpiry:   tokenExpiry,
        refreshExpiry: refreshExpiry,
    }
}

func (j *JWTService) GenerateTokenPair(userID, email string, roles []string) (string, string, error) {
    // Generate access token
    accessClaims := &Claims{
        UserID: userID,
        Email:  email,
        Roles:  roles,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    j.issuer,
            Subject:   userID,
            Audience:  []string{"erpgo-api"},
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.tokenExpiry)),
            NotBefore: jwt.NewNumericDate(time.Now()),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ID:        generateJTI(),
        },
    }

    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    accessTokenString, err := accessToken.SignedString(j.secretKey)
    if err != nil {
        return "", "", err
    }

    // Generate refresh token
    refreshClaims := &jwt.RegisteredClaims{
        Issuer:    j.issuer,
        Subject:   userID,
        Audience:  []string{"erpgo-refresh"},
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.refreshExpiry)),
        NotBefore: jwt.NewNumericDate(time.Now()),
        IssuedAt:  jwt.NewNumericDate(time.Now()),
        ID:        generateJTI(),
    }

    refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    refreshTokenString, err := refreshToken.SignedString(j.refreshSecret)
    if err != nil {
        return "", "", err
    }

    return accessTokenString, refreshTokenString, nil
}

func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return j.secretKey, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("invalid token")
}

func generateJTI() string {
    bytes := make([]byte, 16)
    rand.Read(bytes)
    return base64.URLEncoding.EncodeToString(bytes)
}
```

#### Token Blacklist Implementation

```go
// pkg/auth/blacklist.go
package auth

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
)

type TokenBlacklist struct {
    redis *redis.Client
}

func NewTokenBlacklist(redisClient *redis.Client) *TokenBlacklist {
    return &TokenBlacklist{redis: redisClient}
}

func (tb *TokenBlacklist) BlacklistToken(jti string, expiry time.Time) error {
    key := fmt.Sprintf("blacklist:%s", jti)
    duration := time.Until(expiry)

    return tb.redis.Set(ctx, key, "1", duration).Err()
}

func (tb *TokenBlacklist) IsTokenBlacklisted(jti string) (bool, error) {
    key := fmt.Sprintf("blacklist:%s", jti)
    result, err := tb.redis.Exists(ctx, key).Result()
    return result > 0, err
}

func (tb *TokenBlacklist) BlacklistUserTokens(userID string) error {
    // This would require tracking active tokens per user
    // Implementation depends on your token storage strategy
    return nil
}
```

### Role-Based Access Control (RBAC)

#### Permission Model

```go
// pkg/auth/rbac.go
package auth

import (
    "context"
    "errors"
    "strings"
)

type Permission struct {
    Resource string `json:"resource"`
    Action   string `json:"action"`
    Effect   string `json:"effect"` // "allow" or "deny"
}

type Role struct {
    ID          string       `json:"id"`
    Name        string       `json:"name"`
    Permissions []Permission `json:"permissions"`
}

type RBACService struct {
    roleRepo    RoleRepository
    userRepo    UserRepository
    cache       CacheService
}

func NewRBACService(roleRepo RoleRepository, userRepo UserRepository, cache CacheService) *RBACService {
    return &RBACService{
        roleRepo: roleRepo,
        userRepo: userRepo,
        cache:    cache,
    }
}

func (r *RBACService) CanAccess(ctx context.Context, userID, resource, action string) (bool, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("rbac:%s:%s:%s", userID, resource, action)
    if cached, err := r.cache.Get(ctx, cacheKey); err == nil {
        return cached.(bool), nil
    }

    // Get user roles
    roles, err := r.getUserRoles(ctx, userID)
    if err != nil {
        return false, err
    }

    // Evaluate permissions
    allowed := r.evaluatePermissions(roles, resource, action)

    // Cache result
    r.cache.Set(ctx, cacheKey, allowed, 5*time.Minute)

    return allowed, nil
}

func (r *RBACService) evaluatePermissions(roles []Role, resource, action string) bool {
    for _, role := range roles {
        for _, permission := range role.Permissions {
            if r.matchesPermission(permission, resource, action) {
                if permission.Effect == "deny" {
                    return false // Deny takes precedence
                }
                if permission.Effect == "allow" {
                    return true
                }
            }
        }
    }
    return false // Default deny
}

func (r *RBACService) matchesPermission(permission Permission, resource, action string) bool {
    // Support wildcards in permissions
    resourceMatch := permission.Resource == "*" || permission.Resource == resource
    actionMatch := permission.Action == "*" || permission.Action == action

    // Support hierarchical resources (e.g., "orders.*" matches "orders.create")
    if strings.HasSuffix(permission.Resource, ".*") {
        baseResource := strings.TrimSuffix(permission.Resource, ".*")
        resourceMatch = strings.HasPrefix(resource, baseResource)
    }

    return resourceMatch && actionMatch
}

// Predefined permissions
var (
    PermissionUserRead   = Permission{Resource: "users", Action: "read", Effect: "allow"}
    PermissionUserWrite  = Permission{Resource: "users", Action: "write", Effect: "allow"}
    PermissionOrderRead  = Permission{Resource: "orders", Action: "read", Effect: "allow"}
    PermissionOrderWrite = Permission{Resource: "orders", Action: "write", Effect: "allow"}
    PermissionProductRead   = Permission{Resource: "products", Action: "read", Effect: "allow"}
    PermissionProductWrite  = Permission{Resource: "products", Action: "write", Effect: "allow"}
    PermissionAdminAll    = Permission{Resource: "*", Action: "*", Effect: "allow"}
)

// Predefined roles
var (
    RoleAdmin = Role{
        ID:          "admin",
        Name:        "Administrator",
        Permissions: []Permission{PermissionAdminAll},
    }

    RoleManager = Role{
        ID:   "manager",
        Name: "Manager",
        Permissions: []Permission{
            PermissionUserRead,
            PermissionOrderRead,
            PermissionOrderWrite,
            PermissionProductRead,
            PermissionProductWrite,
        },
    }

    RoleUser = Role{
        ID:   "user",
        Name: "User",
        Permissions: []Permission{
            PermissionOrderRead,
            PermissionProductRead,
        },
    }
)
```

## Data Protection

### Encryption at Rest

#### File Encryption Service

```go
// pkg/encryption/file.go
package encryption

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "fmt"
    "io"
)

type FileEncryptionService struct {
    masterKey []byte
}

func NewFileEncryptionService(masterKey []byte) *FileEncryptionService {
    return &FileEncryptionService{masterKey: masterKey}
}

func (f *FileEncryptionService) EncryptFile(src, dst string) error {
    data, err := os.ReadFile(src)
    if err != nil {
        return err
    }

    encrypted, err := f.Encrypt(data)
    if err != nil {
        return err
    }

    return os.WriteFile(dst, encrypted, 0644)
}

func (f *FileEncryptionService) DecryptFile(src, dst string) error {
    encrypted, err := os.ReadFile(src)
    if err != nil {
        return err
    }

    decrypted, err := f.Decrypt(encrypted)
    if err != nil {
        return err
    }

    return os.WriteFile(dst, decrypted, 0644)
}

func (f *FileEncryptionService) Encrypt(data []byte) ([]byte, error) {
    block, err := aes.NewCipher(f.masterKey)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err = rand.Read(nonce); err != nil {
        return nil, err
    }

    ciphertext := gcm.Seal(nonce, nonce, data, nil)
    return ciphertext, nil
}

func (f *FileEncryptionService) Decrypt(data []byte) ([]byte, error) {
    block, err := aes.NewCipher(f.masterKey)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return nil, errors.New("ciphertext too short")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    return gcm.Open(nil, nonce, ciphertext, nil)
}
```

#### Database Encryption

```go
// pkg/encryption/database.go
package encryption

import (
    "database/sql/driver"
    "encoding/json"
    "errors"
    "fmt"
)

type EncryptedData struct {
    Data  string `json:"data"`
    Nonce string `json:"nonce"`
}

type EncryptedField struct {
    Value *EncryptedData
}

func (e *EncryptedField) Scan(value interface{}) error {
    if value == nil {
        return nil
    }

    switch v := value.(type) {
    case []byte:
        return json.Unmarshal(v, &e.Value)
    case string:
        return json.Unmarshal([]byte(v), &e.Value)
    default:
        return fmt.Errorf("cannot scan %T into EncryptedField", value)
    }
}

func (e EncryptedField) Value() (driver.Value, error) {
    if e.Value == nil {
        return nil, nil
    }
    return json.Marshal(e.Value)
}

// Database encryption service
type DatabaseEncryption struct {
    key []byte
}

func NewDatabaseEncryption(key []byte) *DatabaseEncryption {
    return &DatabaseEncryption{key: key}
}

func (d *DatabaseEncryption) EncryptField(plaintext string) (*EncryptedData, error) {
    block, err := aes.NewCipher(d.key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err = rand.Read(nonce); err != nil {
        return nil, err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

    return &EncryptedData{
        Data:  base64.StdEncoding.EncodeToString(ciphertext),
        Nonce: base64.StdEncoding.EncodeToString(nonce),
    }, nil
}

func (d *DatabaseEncryption) DecryptField(encrypted *EncryptedData) (string, error) {
    if encrypted == nil {
        return "", nil
    }

    ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Data)
    if err != nil {
        return "", err
    }

    nonce, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(d.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

### Data Masking

#### PII Data Masking

```go
// pkg/privacy/masking.go
package privacy

import (
    "strings"
    "unicode"
)

type DataMasker struct {
    maskChar rune
}

func NewDataMasker(maskChar rune) *DataMasker {
    return &DataMasker{maskChar: maskChar}
}

func (d *DataMasker) MaskEmail(email string) string {
    if email == "" {
        return ""
    }

    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return email
    }

    username := parts[0]
    domain := parts[1]

    if len(username) <= 2 {
        return strings.Repeat(string(d.maskChar), len(username)) + "@" + domain
    }

    maskedUsername := string(username[0]) +
        strings.Repeat(string(d.maskChar), len(username)-2) +
        string(username[len(username)-1])

    return maskedUsername + "@" + domain
}

func (d *DataMasker) MaskPhone(phone string) string {
    if phone == "" {
        return ""
    }

    // Remove non-digit characters
    var digits []rune
    for _, r := range phone {
        if unicode.IsDigit(r) {
            digits = append(digits, r)
        }
    }

    if len(digits) <= 4 {
        return strings.Repeat(string(d.maskChar), len(digits))
    }

    // Show last 4 digits, mask the rest
    masked := strings.Repeat(string(d.maskChar), len(digits)-4) + string(digits[len(digits)-4:])

    return masked
}

func (d *DataMasker) MaskCreditCard(card string) string {
    if card == "" {
        return ""
    }

    // Remove spaces and dashes
    cleaned := strings.ReplaceAll(strings.ReplaceAll(card, " ", ""), "-", "")

    if len(cleaned) != 16 {
        return strings.Repeat(string(d.maskChar), len(card))
    }

    // Show last 4 digits
    return strings.Repeat(string(d.maskChar), 12) + cleaned[12:]
}

func (d *DataMasker) MaskSSN(ssn string) string {
    if ssn == "" {
        return ""
    }

    // Remove non-digit characters
    var digits []rune
    for _, r := range ssn {
        if unicode.IsDigit(r) {
            digits = append(digits, r)
        }
    }

    if len(digits) != 9 {
        return strings.Repeat(string(d.maskChar), len(ssn))
    }

    // Format: XXX-XX-XXXX
    return string(digits[0]) + string(digits[1]) + string(digits[2]) + "-" +
           strings.Repeat(string(d.maskChar), 2) + "-" +
           string(digits[5:]) + string(digits[6:]) + string(digits[7:]) + string(digits[8])
}
```

## Network Security

### TLS/SSL Configuration

#### Secure TLS Configuration

```go
// pkg/tls/config.go
package tls

import (
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
)

func LoadTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
    // Load certificate and key
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, err
    }

    // Create TLS config
    config := &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
            tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
            tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
        },
        PreferServerCipherSuites: true,
        CurvePreferences: []tls.CurveID{
            tls.X25519,
            tls.CurveP256,
        },
    }

    // Load CA certificate if provided
    if caFile != "" {
        caCert, err := ioutil.ReadFile(caFile)
        if err != nil {
            return nil, err
        }

        caCertPool := x509.NewCertPool()
        caCertPool.AppendCertsFromPEM(caCert)
        config.RootCAs = caCertPool
        config.ClientCAs = caCertPool
        config.ClientAuth = tls.RequireAndVerifyClientCert
    }

    return config, nil
}
```

### CORS Security

#### Secure CORS Configuration

```go
// pkg/middleware/cors.go
package middleware

import (
    "strings"
    "github.com/gin-gonic/gin"
)

type CORSConfig struct {
    AllowedOrigins   []string
    AllowedMethods   []string
    AllowedHeaders   []string
    ExposedHeaders   []string
    AllowCredentials bool
    MaxAge           int
}

func SecureCORS(config CORSConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        origin := c.Request.Header.Get("Origin")

        // Check if origin is allowed
        allowed := false
        for _, allowedOrigin := range config.AllowedOrigins {
            if allowedOrigin == "*" || allowedOrigin == origin {
                allowed = true
                break
            }
        }

        if allowed {
            c.Header("Access-Control-Allow-Origin", origin)
        }

        if len(config.AllowedMethods) > 0 {
            c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
        }

        if len(config.AllowedHeaders) > 0 {
            c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
        }

        if len(config.ExposedHeaders) > 0 {
            c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
        }

        if config.AllowCredentials {
            c.Header("Access-Control-Allow-Credentials", "true")
        }

        if config.MaxAge > 0 {
            c.Header("Access-Control-Max-Age", string(rune(config.MaxAge)))
        }

        // Handle preflight requests
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}

// Secure CORS configuration for production
var ProductionCORSConfig = CORSConfig{
    AllowedOrigins: []string{
        "https://app.erpgo.example.com",
        "https://admin.erpgo.example.com",
    },
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders: []string{
        "Origin",
        "Content-Type",
        "Accept",
        "Authorization",
        "X-Requested-With",
    },
    ExposedHeaders:   []string{"X-Total-Count"},
    AllowCredentials: true,
    MaxAge:           86400, // 24 hours
}
```

## Application Security

### Input Validation

#### Comprehensive Input Validation

```go
// pkg/validation/validator.go
package validation

import (
    "errors"
    "fmt"
    "net/mail"
    "regexp"
    "strings"
    "unicode"
)

type Validator struct {
    rules map[string][]ValidationRule
}

type ValidationRule struct {
    Name      string
    Validator func(interface{}) error
    Message   string
}

func NewValidator() *Validator {
    return &Validator{
        rules: make(map[string][]ValidationRule),
    }
}

func (v *Validator) AddRule(field string, rule ValidationRule) {
    v.rules[field] = append(v.rules[field], rule)
}

func (v *Validator) Validate(data map[string]interface{}) error {
    for field, rules := range v.rules {
        value, exists := data[field]
        if !exists {
            continue
        }

        for _, rule := range rules {
            if err := rule.Validator(value); err != nil {
                return fmt.Errorf("%s: %s", field, rule.Message)
            }
        }
    }
    return nil
}

// Common validation rules
var (
    RuleRequired = ValidationRule{
        Name: "required",
        Validator: func(value interface{}) error {
            if value == nil || value == "" {
                return errors.New("field is required")
            }
            return nil
        },
        Message: "field is required",
    }

    RuleEmail = ValidationRule{
        Name: "email",
        Validator: func(value interface{}) error {
            email, ok := value.(string)
            if !ok {
                return errors.New("must be a string")
            }
            _, err := mail.ParseAddress(email)
            return err
        },
        Message: "must be a valid email address",
    }

    RuleMinLength(min int) ValidationRule {
        return ValidationRule{
            Name: fmt.Sprintf("min_length_%d", min),
            Validator: func(value interface{}) error {
                str, ok := value.(string)
                if !ok {
                    return errors.New("must be a string")
                }
                if len(str) < min {
                    return fmt.Errorf("must be at least %d characters", min)
                }
                return nil
            },
            Message: fmt.Sprintf("must be at least %d characters", min),
        }
    }

    RuleMaxLength(max int) ValidationRule {
        return ValidationRule{
            Name: fmt.Sprintf("max_length_%d", max),
            Validator: func(value interface{}) error {
                str, ok := value.(string)
                if !ok {
                    return errors.New("must be a string")
                }
                if len(str) > max {
                    return fmt.Errorf("must be no more than %d characters", max)
                }
                return nil
            },
            Message: fmt.Sprintf("must be no more than %d characters", max),
        }
    }

    RuleUUID = ValidationRule{
        Name: "uuid",
        Validator: func(value interface{}) error {
            str, ok := value.(string)
            if !ok {
                return errors.New("must be a string")
            }
            uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
            if !uuidRegex.MatchString(str) {
                return errors.New("must be a valid UUID")
            }
            return nil
        },
        Message: "must be a valid UUID",
    }

    RuleNoHTML = ValidationRule{
        Name: "no_html",
        Validator: func(value interface{}) error {
            str, ok := value.(string)
            if !ok {
                return errors.New("must be a string")
            }
            htmlRegex := regexp.MustCompile(`<[^>]*>`)
            if htmlRegex.MatchString(str) {
                return errors.New("must not contain HTML tags")
            }
            return nil
        },
        Message: "must not contain HTML tags",
    }

    RuleSQLInjection = ValidationRule{
        Name: "no_sql_injection",
        Validator: func(value interface{}) error {
            str, ok := value.(string)
            if !ok {
                return errors.New("must be a string")
            }

            lowerStr := strings.ToLower(str)
            dangerousPatterns := []string{
                "drop table", "delete from", "insert into", "update set",
                "union select", "exec(", "script>", "javascript:",
                "onerror=", "onload=", "eval(",
            }

            for _, pattern := range dangerousPatterns {
                if strings.Contains(lowerStr, pattern) {
                    return errors.New("contains potentially dangerous content")
                }
            }
            return nil
        },
        Message: "contains potentially dangerous content",
    }
)

// Sanitization functions
func SanitizeHTML(input string) string {
    // Remove HTML tags
    re := regexp.MustCompile(`<[^>]*>`)
    return re.ReplaceAllString(input, "")
}

func SanitizeSQL(input string) string {
    // Remove dangerous SQL characters
    dangerous := []string{"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_"}
    result := input
    for _, char := range dangerous {
        result = strings.ReplaceAll(result, char, "")
    }
    return result
}
```

### XSS Protection

```go
// pkg/security/xss.go
package security

import (
    "html"
    "strings"
    "unicode"
)

type XSSProtection struct {
    allowedTags    []string
    allowedAttrs   []string
    stripAllHTML  bool
}

func NewXSSProtection(stripAllHTML bool) *XSSProtection {
    return &XSSProtection{
        allowedTags: []string{"p", "br", "strong", "em", "u"},
        allowedAttrs: []string{},
        stripAllHTML: stripAllHTML,
    }
}

func (x *XSSProtection) Sanitize(input string) string {
    if x.stripAllHTML {
        return html.EscapeString(input)
    }

    // Custom HTML sanitization
    return x.sanitizeHTML(input)
}

func (x *XSSProtection) sanitizeHTML(input string) string {
    var result strings.Builder
    var currentTag strings.Builder
    var inTag bool
    var tagName strings.Builder

    for i, r := range input {
        if r == '<' {
            if !inTag {
                inTag = true
                currentTag.Reset()
                tagName.Reset()
            }
            continue
        }

        if r == '>' && inTag {
            inTag = false
            tagContent := currentTag.String()

            if x.isAllowedTag(tagContent) {
                result.WriteString("<" + tagContent + ">")
            }
            continue
        }

        if inTag {
            currentTag.WriteRune(r)

            // Extract tag name (first word after <)
            if tagName.Len() == 0 && unicode.IsLetter(r) {
                tagName.WriteRune(r)
            } else if tagName.Len() > 0 && unicode.IsLetter(r) {
                tagName.WriteRune(r)
            }
            continue
        }

        // Escape characters outside of tags
        switch r {
        case '&':
            result.WriteString("&amp;")
        case '<':
            result.WriteString("&lt;")
        case '>':
            result.WriteString("&gt;")
        case '"':
            result.WriteString("&quot;")
        case '\'':
            result.WriteString("&#x27;")
        default:
            result.WriteRune(r)
        }
    }

    return result.String()
}

func (x *XSSProtection) isAllowedTag(tagContent string) bool {
    // Extract tag name
    parts := strings.Fields(tagContent)
    if len(parts) == 0 {
        return false
    }

    tagName := strings.ToLower(parts[0])
    tagName = strings.Trim(tagName, "/")

    for _, allowed := range x.allowedTags {
        if tagName == allowed {
            return true
        }
    }

    return false
}
```

## Infrastructure Security

### Container Security

#### Secure Docker Configuration

```dockerfile
# Dockerfile.secure
FROM golang:1.21-alpine AS builder

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy and build
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o erpgo cmd/api/main.go

# Runtime stage
FROM alpine:latest

# Install security updates
RUN apk update && apk upgrade && \
    apk add --no-cache ca-certificates tzdata curl && \
    rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary
COPY --from=builder /app/erpgo .

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Run application
CMD ["./erpgo"]
```

#### Docker Security Configuration

```yaml
# docker-compose.security.yml
version: '3.8'

services:
  erpgo-api:
    build:
      context: .
      dockerfile: Dockerfile.secure
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp:noexec,nosuid,size=100m
    user: "1001:1001"
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
    environment:
      - GIN_MODE=release
    networks:
      - erpgo-network

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: erp
      POSTGRES_USER: erpgo
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data:rw
      - ./postgres/postgresql.conf:/etc/postgresql/postgresql.conf:ro
    security_opt:
      - no-new-privileges:true
    user: "999:999"
    networks:
      - erpgo-network

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    security_opt:
      - no-new-privileges:true
    user: "999:999"
    tmpfs:
      - /tmp:noexec,nosuid,size=10m
    networks:
      - erpgo-network

networks:
  erpgo-network:
    driver: bridge
    internal: true
    ipam:
      config:
        - subnet: 172.20.0.0/16

volumes:
  postgres_data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /data/postgres
```

### Kubernetes Security

#### Pod Security Policy

```yaml
# k8s/security/pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: erpgo-restricted-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: true
  securityContext:
    runAsNonRoot: true
    runAsUser: 1001
    fsGroup: 1001
```

#### Network Policy

```yaml
# k8s/security/network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: erpgo-network-policy
spec:
  podSelector:
    matchLabels:
      app: erpgo-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

## Database Security

### PostgreSQL Security Configuration

```sql
-- postgresql-security.conf

# SSL Configuration
ssl = on
ssl_cert_file = '/var/lib/postgresql/server.crt'
ssl_key_file = '/var/lib/postgresql/server.key'
ssl_ca_file = '/var/lib/postgresql/ca.crt'

# Authentication
password_encryption = scram-sha-256
logging_collector = on
log_directory = 'pg_log'
log_filename = 'postgresql-%Y-%m-%d_%H%M%S.log'
log_statement = 'all'
log_min_duration_statement = 1000

# Row Level Security
ALTER DATABASE erp SET row_security = on;

-- Create application user with limited privileges
CREATE ROLE erpgo_app WITH LOGIN PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE erp TO erpgo_app;
GRANT USAGE ON SCHEMA public TO erpgo_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO erpgo_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO erpgo_app;

-- Row Level Security policies
CREATE POLICY user_isolation_policy ON users
    FOR ALL
    TO erpgo_app
    USING (id = current_setting('app.current_user_id')::uuid);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Audit trigger for sensitive operations
CREATE OR REPLACE FUNCTION audit_trigger_function()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        INSERT INTO audit_logs (table_name, operation, old_data, user_id, timestamp)
        VALUES (TG_TABLE_NAME, TG_OP, row_to_json(OLD), current_setting('app.current_user_id')::uuid, NOW());
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_logs (table_name, operation, old_data, new_data, user_id, timestamp)
        VALUES (TG_TABLE_NAME, TG_OP, row_to_json(OLD), row_to_json(NEW), current_setting('app.current_user_id')::uuid, NOW());
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO audit_logs (table_name, operation, new_data, user_id, timestamp)
        VALUES (TG_TABLE_NAME, TG_OP, row_to_json(NEW), current_setting('app.current_user_id')::uuid, NOW());
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Apply audit trigger to sensitive tables
CREATE TRIGGER users_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_function();
```

## API Security

### API Key Management

```go
// pkg/auth/apikey.go
package auth

import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "time"
)

type APIKey struct {
    ID        string    `json:"id"`
    Key       string    `json:"key"`
    Name      string    `json:"name"`
    Scopes    []string  `json:"scopes"`
    CreatedAt time.Time `json:"created_at"`
    ExpiresAt time.Time `json:"expires_at"`
    LastUsed  time.Time `json:"last_used"`
}

type APIKeyService struct {
    store APIKeyStore
}

func NewAPIKeyService(store APIKeyStore) *APIKeyService {
    return &APIKeyService{store: store}
}

func (a *APIKeyService) GenerateAPIKey(name string, scopes []string, expiry time.Duration) (*APIKey, error) {
    // Generate secure random key
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return nil, err
    }

    key := "erpgo_" + base64.URLEncoding.EncodeToString(bytes)

    apiKey := &APIKey{
        ID:        generateID(),
        Key:       key,
        Name:      name,
        Scopes:    scopes,
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(expiry),
    }

    if err := a.store.Create(apiKey); err != nil {
        return nil, err
    }

    return apiKey, nil
}

func (a *APIKeyService) ValidateAPIKey(key string) (*APIKey, error) {
    apiKey, err := a.store.FindByKey(key)
    if err != nil {
        return nil, err
    }

    if apiKey.ExpiresAt.Before(time.Now()) {
        return nil, fmt.Errorf("API key expired")
    }

    // Update last used time
    apiKey.LastUsed = time.Now()
    a.store.Update(apiKey)

    return apiKey, nil
}

func (a *APIKeyService) HasScope(apiKey *APIKey, requiredScope string) bool {
    for _, scope := range apiKey.Scopes {
        if scope == requiredScope || scope == "*" {
            return true
        }
    }
    return false
}

// API Key middleware
func APIKeyMiddleware(apiKeyService *APIKeyService) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Missing API key"})
            c.Abort()
            return
        }

        // Extract API key from "Bearer erpgo_..." format
        if !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(401, gin.H{"error": "Invalid authorization header format"})
            c.Abort()
            return
        }

        key := strings.TrimPrefix(authHeader, "Bearer ")

        apiKey, err := apiKeyService.ValidateAPIKey(key)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid API key"})
            c.Abort()
            return
        }

        // Store API key in context for use in handlers
        c.Set("api_key", apiKey)
        c.Next()
    }
}
```

## Logging & Monitoring

### Security Event Logging

```go
// pkg/security/audit.go
package security

import (
    "context"
    "encoding/json"
    "time"
)

type SecurityEvent struct {
    ID        string                 `json:"id"`
    Timestamp time.Time              `json:"timestamp"`
    Type      string                 `json:"type"`
    UserID    string                 `json:"user_id,omitempty"`
    IPAddress string                 `json:"ip_address"`
    UserAgent string                 `json:"user_agent"`
    Resource  string                 `json:"resource,omitempty"`
    Action    string                 `json:"action,omitempty"`
    Success   bool                   `json:"success"`
    Details   map[string]interface{} `json:"details,omitempty"`
    Risk      string                 `json:"risk"` // low, medium, high, critical
}

type AuditLogger struct {
    logger Logger
}

func NewAuditLogger(logger Logger) *AuditLogger {
    return &AuditLogger{logger: logger}
}

func (a *AuditLogger) LogEvent(ctx context.Context, event SecurityEvent) {
    event.ID = generateEventID()
    event.Timestamp = time.Now()

    // Add context information
    if userID, exists := ctx.Value("user_id").(string); exists {
        event.UserID = userID
    }

    if ip, exists := ctx.Value("ip_address").(string); exists {
        event.IPAddress = ip
    }

    if userAgent, exists := ctx.Value("user_agent").(string); exists {
        event.UserAgent = userAgent
    }

    // Log the event
    a.logger.Info().
        Str("event_id", event.ID).
        Str("event_type", event.Type).
        Str("user_id", event.UserID).
        Str("ip_address", event.IPAddress).
        Str("resource", event.Resource).
        Str("action", event.Action).
        Bool("success", event.Success).
        Str("risk", event.Risk).
        Interface("details", event.Details).
        Msg("Security event")
}

// Security event types
const (
    EventLogin           = "login"
    EventLogout          = "logout"
    EventPasswordReset   = "password_reset"
    EventPermissionDeny = "permission_deny"
    EventDataAccess     = "data_access"
    EventDataModification = "data_modification"
    EventAPIKeyAccess   = "api_key_access"
    EventSuspiciousActivity = "suspicious_activity"
)

// Risk levels
const (
    RiskLow      = "low"
    RiskMedium   = "medium"
    RiskHigh     = "high"
    RiskCritical = "critical"
)
```

## Incident Response

### Security Incident Response

```go
// pkg/security/incident.go
package security

import (
    "context"
    "fmt"
    "time"
)

type Incident struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Severity    string    `json:"severity"`
    Status      string    `json:"status"`
    DetectedAt  time.Time `json:"detected_at"`
    AssignedTo  string    `json:"assigned_to,omitempty"`
    ResolvedAt  time.Time `json:"resolved_at,omitempty"`
    Actions     []string  `json:"actions,omitempty"`
}

type IncidentResponse struct {
    incident *Incident
    notifier Notifier
    store    IncidentStore
}

func NewIncidentResponse(notifier Notifier, store IncidentStore) *IncidentResponse {
    return &IncidentResponse{
        notifier: notifier,
        store:    store,
    }
}

func (ir *IncidentResponse) CreateIncident(title, description, severity string) (*Incident, error) {
    incident := &Incident{
        ID:          generateIncidentID(),
        Title:       title,
        Description: description,
        Severity:    severity,
        Status:      "open",
        DetectedAt:  time.Now(),
    }

    if err := ir.store.Create(incident); err != nil {
        return nil, err
    }

    // Notify appropriate teams
    if err := ir.notifier.NotifyIncident(incident); err != nil {
        // Log error but don't fail the incident creation
        log.Printf("Failed to notify incident: %v", err)
    }

    return incident, nil
}

func (ir *IncidentResponse) ResolveIncident(incidentID string, resolution string) error {
    incident, err := ir.store.Get(incidentID)
    if err != nil {
        return err
    }

    incident.Status = "resolved"
    incident.ResolvedAt = time.Now()

    if err := ir.store.Update(incident); err != nil {
        return err
    }

    // Notify resolution
    return ir.notifier.NotifyResolution(incident, resolution)
}
```

## Security Checklist

### Development Security Checklist

- [ ] Input validation implemented for all user inputs
- [ ] Output encoding to prevent XSS
- [ ] SQL injection prevention (parameterized queries)
- [ ] CSRF protection for state-changing operations
- [ ] Secure session management
- [ ] Password complexity requirements
- [ ] Rate limiting implemented
- [ ] Security headers configured
- [ ] Error handling doesn't leak sensitive information
- [ ] Dependencies regularly updated
- [ ] Static code analysis tools integrated
- [ ] Security testing in CI/CD pipeline

### Infrastructure Security Checklist

- [ ] Firewalls configured with whitelist approach
- [ ] SSL/TLS properly configured
- [ ] Container security best practices implemented
- [ ] Least privilege access for all services
- [ ] Regular security scans performed
- [ ] Backup encryption enabled
- [ ] Log aggregation and monitoring in place
- [ ] Network segmentation implemented
- [ ] DDoS protection configured
- [ ] Intrusion detection system deployed
- [ ] Security incident response plan documented
- [ ] Regular penetration testing conducted

### Operational Security Checklist

- [ ] Access reviews performed quarterly
- [ ] Security awareness training for all staff
- [ ] Incident response drills conducted
- [ ] Security metrics tracked and reported
- [ ] Vulnerability management process in place
- [ ] Change management includes security review
- [ ] Disaster recovery plan tested regularly
- [ ] Data retention policies implemented
- [ ] Privacy regulations compliance verified
- [ ] Third-party security assessments completed
- [ ] Security documentation maintained
- [ ] Continuous security monitoring active

---

**Note**: Security is an ongoing process, not a one-time implementation. Regular reviews, updates, and improvements are essential to maintain the security posture of the ERPGo system.