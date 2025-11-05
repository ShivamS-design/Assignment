package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// OfflineAuthProvider supports offline installations without external dependencies
type OfflineAuthProvider struct {
	userStore    *FileUserStore
	sessionStore *FileSessionStore
	jwtSecret    []byte
	config       OfflineConfig
}

type OfflineConfig struct {
	DataDir         string        `json:"data_dir"`
	SessionTimeout  time.Duration `json:"session_timeout"`
	PasswordPolicy  PasswordPolicy `json:"password_policy"`
	MaxLoginAttempts int          `json:"max_login_attempts"`
	LockoutDuration time.Duration `json:"lockout_duration"`
}

type PasswordPolicy struct {
	MinLength    int  `json:"min_length"`
	RequireUpper bool `json:"require_upper"`
	RequireLower bool `json:"require_lower"`
	RequireDigit bool `json:"require_digit"`
	RequireSpecial bool `json:"require_special"`
}

type FileUserStore struct {
	filePath string
	users    map[string]*OfflineUser
	attempts map[string]*LoginAttempts
}

type OfflineUser struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Role         Role      `json:"role"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login"`
	Salt         string    `json:"salt"`
}

type LoginAttempts struct {
	Count     int       `json:"count"`
	LastAttempt time.Time `json:"last_attempt"`
	LockedUntil time.Time `json:"locked_until"`
}

type FileSessionStore struct {
	filePath string
	sessions map[string]*OfflineSession
}

type OfflineSession struct {
	ID        string    `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Active    bool      `json:"active"`
}

func NewOfflineAuthProvider(config OfflineConfig) (*OfflineAuthProvider, error) {
	if err := os.MkdirAll(config.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Generate or load JWT secret
	secretPath := filepath.Join(config.DataDir, "jwt.secret")
	jwtSecret, err := loadOrGenerateSecret(secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to setup JWT secret: %w", err)
	}

	userStore, err := NewFileUserStore(filepath.Join(config.DataDir, "users.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user store: %w", err)
	}

	sessionStore, err := NewFileSessionStore(filepath.Join(config.DataDir, "sessions.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session store: %w", err)
	}

	provider := &OfflineAuthProvider{
		userStore:    userStore,
		sessionStore: sessionStore,
		jwtSecret:    jwtSecret,
		config:       config,
	}

	// Create default admin user if none exists
	if err := provider.ensureDefaultAdmin(); err != nil {
		return nil, fmt.Errorf("failed to create default admin: %w", err)
	}

	return provider, nil
}

func (p *OfflineAuthProvider) Authenticate(username, password, ipAddress string) (*OfflineUser, error) {
	// Check login attempts
	if p.isAccountLocked(username) {
		return nil, fmt.Errorf("account temporarily locked due to too many failed attempts")
	}

	user, err := p.userStore.GetUser(username)
	if err != nil {
		p.recordFailedAttempt(username)
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.Active {
		return nil, fmt.Errorf("account disabled")
	}

	// Verify password
	if err := p.verifyPassword(password, user.PasswordHash, user.Salt); err != nil {
		p.recordFailedAttempt(username)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset failed attempts on successful login
	p.resetFailedAttempts(username)
	
	// Update last login
	user.LastLogin = time.Now()
	p.userStore.UpdateUser(user)

	return user, nil
}

func (p *OfflineAuthProvider) CreateSession(user *OfflineUser, ipAddress, userAgent string) (*OfflineSession, error) {
	sessionID := generateSecureID()
	session := &OfflineSession{
		ID:        sessionID,
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(p.config.SessionTimeout),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Active:    true,
	}

	return session, p.sessionStore.SaveSession(session)
}

func (p *OfflineAuthProvider) ValidateSession(sessionID string) (*OfflineSession, error) {
	session, err := p.sessionStore.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if !session.Active || time.Now().After(session.ExpiresAt) {
		session.Active = false
		p.sessionStore.SaveSession(session)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

func (p *OfflineAuthProvider) CreateUser(username, email, password string, role Role) error {
	if err := p.validatePassword(password); err != nil {
		return err
	}

	if _, err := p.userStore.GetUser(username); err == nil {
		return fmt.Errorf("user already exists")
	}

	salt := generateSalt()
	passwordHash, err := p.hashPassword(password, salt)
	if err != nil {
		return err
	}

	user := &OfflineUser{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		Active:       true,
		CreatedAt:    time.Now(),
		Salt:         salt,
	}

	return p.userStore.SaveUser(user)
}

func (p *OfflineAuthProvider) ensureDefaultAdmin() error {
	if _, err := p.userStore.GetUser("admin"); err == nil {
		return nil // Admin already exists
	}

	return p.CreateUser("admin", "admin@localhost", "admin123", RoleAdmin)
}

func (p *OfflineAuthProvider) validatePassword(password string) error {
	policy := p.config.PasswordPolicy
	
	if len(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters", policy.MinLength)
	}

	if policy.RequireUpper && !containsUpper(password) {
		return fmt.Errorf("password must contain uppercase letter")
	}

	if policy.RequireLower && !containsLower(password) {
		return fmt.Errorf("password must contain lowercase letter")
	}

	if policy.RequireDigit && !containsDigit(password) {
		return fmt.Errorf("password must contain digit")
	}

	if policy.RequireSpecial && !containsSpecial(password) {
		return fmt.Errorf("password must contain special character")
	}

	return nil
}

func (p *OfflineAuthProvider) hashPassword(password, salt string) (string, error) {
	// Combine password with salt
	combined := password + salt
	hash, err := bcrypt.GenerateFromPassword([]byte(combined), bcrypt.DefaultCost)
	return string(hash), err
}

func (p *OfflineAuthProvider) verifyPassword(password, hash, salt string) error {
	combined := password + salt
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(combined))
}

func (p *OfflineAuthProvider) isAccountLocked(username string) bool {
	attempts, exists := p.userStore.attempts[username]
	if !exists {
		return false
	}

	return attempts.Count >= p.config.MaxLoginAttempts && 
		   time.Now().Before(attempts.LockedUntil)
}

func (p *OfflineAuthProvider) recordFailedAttempt(username string) {
	attempts, exists := p.userStore.attempts[username]
	if !exists {
		attempts = &LoginAttempts{}
		p.userStore.attempts[username] = attempts
	}

	attempts.Count++
	attempts.LastAttempt = time.Now()
	
	if attempts.Count >= p.config.MaxLoginAttempts {
		attempts.LockedUntil = time.Now().Add(p.config.LockoutDuration)
	}
}

func (p *OfflineAuthProvider) resetFailedAttempts(username string) {
	delete(p.userStore.attempts, username)
}

func NewFileUserStore(filePath string) (*FileUserStore, error) {
	store := &FileUserStore{
		filePath: filePath,
		users:    make(map[string]*OfflineUser),
		attempts: make(map[string]*LoginAttempts),
	}

	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

func (fs *FileUserStore) load() error {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		return err
	}

	var stored struct {
		Users    map[string]*OfflineUser    `json:"users"`
		Attempts map[string]*LoginAttempts `json:"attempts"`
	}

	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	fs.users = stored.Users
	fs.attempts = stored.Attempts
	return nil
}

func (fs *FileUserStore) save() error {
	stored := struct {
		Users    map[string]*OfflineUser    `json:"users"`
		Attempts map[string]*LoginAttempts `json:"attempts"`
	}{
		Users:    fs.users,
		Attempts: fs.attempts,
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, data, 0600)
}

func (fs *FileUserStore) GetUser(username string) (*OfflineUser, error) {
	user, exists := fs.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (fs *FileUserStore) SaveUser(user *OfflineUser) error {
	fs.users[user.Username] = user
	return fs.save()
}

func (fs *FileUserStore) UpdateUser(user *OfflineUser) error {
	return fs.SaveUser(user)
}

func NewFileSessionStore(filePath string) (*FileSessionStore, error) {
	store := &FileSessionStore{
		filePath: filePath,
		sessions: make(map[string]*OfflineSession),
	}

	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

func (fs *FileSessionStore) load() error {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &fs.sessions)
}

func (fs *FileSessionStore) save() error {
	data, err := json.MarshalIndent(fs.sessions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, data, 0600)
}

func (fs *FileSessionStore) GetSession(sessionID string) (*OfflineSession, error) {
	session, exists := fs.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (fs *FileSessionStore) SaveSession(session *OfflineSession) error {
	fs.sessions[session.ID] = session
	return fs.save()
}

func loadOrGenerateSecret(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}

	// Generate new secret
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString(secret)
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return nil, err
	}

	return []byte(encoded), nil
}

func generateSalt() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

func generateSecureID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	hash := sha256.Sum256(bytes)
	return base64.URLEncoding.EncodeToString(hash[:])
}

func containsUpper(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func containsLower(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func containsSpecial(s string) bool {
	special := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	for _, r := range s {
		for _, sp := range special {
			if r == sp {
				return true
			}
		}
	}
	return false
}

func GetDefaultOfflineConfig() OfflineConfig {
	return OfflineConfig{
		DataDir:        "./data/auth",
		SessionTimeout: 24 * time.Hour,
		PasswordPolicy: PasswordPolicy{
			MinLength:      8,
			RequireUpper:   true,
			RequireLower:   true,
			RequireDigit:   true,
			RequireSpecial: false,
		},
		MaxLoginAttempts: 5,
		LockoutDuration:  15 * time.Minute,
	}
}