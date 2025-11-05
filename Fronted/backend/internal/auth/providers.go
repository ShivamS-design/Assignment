package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthProvider interface {
	Authenticate(ctx context.Context, credentials Credentials) (*User, error)
	ValidateToken(token string) (*Claims, error)
	RefreshToken(refreshToken string) (*TokenPair, error)
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Provider string `json:"provider"`
	Token    string `json:"token,omitempty"`
}

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     Role      `json:"role"`
	Provider string    `json:"provider"`
	Active   bool      `json:"active"`
	LastLogin time.Time `json:"last_login"`
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Role     Role      `json:"role"`
	Provider string    `json:"provider"`
	jwt.RegisteredClaims
}

type LocalProvider struct {
	users     map[string]*User
	passwords map[string]string
	jwtSecret []byte
}

func NewLocalProvider(jwtSecret string) *LocalProvider {
	provider := &LocalProvider{
		users:     make(map[string]*User),
		passwords: make(map[string]string),
		jwtSecret: []byte(jwtSecret),
	}
	provider.initDefaultUsers()
	return provider
}

func (p *LocalProvider) initDefaultUsers() {
	// Create default admin user
	adminID := uuid.New()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	
	p.users["admin"] = &User{
		ID:       adminID,
		Username: "admin",
		Email:    "admin@localhost",
		Role:     RoleAdmin,
		Provider: "local",
		Active:   true,
	}
	p.passwords["admin"] = string(hashedPassword)
}

func (p *LocalProvider) Authenticate(ctx context.Context, creds Credentials) (*User, error) {
	user, exists := p.users[creds.Username]
	if !exists || !user.Active {
		return nil, fmt.Errorf("invalid credentials")
	}

	hashedPassword := p.passwords[creds.Username]
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(creds.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	user.LastLogin = time.Now()
	return user, nil
}

func (p *LocalProvider) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return p.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (p *LocalProvider) RefreshToken(refreshToken string) (*TokenPair, error) {
	claims, err := p.ValidateToken(refreshToken)
	if err != nil {
		return nil, err
	}

	user, exists := p.users[claims.Username]
	if !exists || !user.Active {
		return nil, fmt.Errorf("user not found or inactive")
	}

	return p.GenerateTokens(user)
}

func (p *LocalProvider) GenerateTokens(user *User) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(15 * time.Minute)
	refreshExpiry := now.Add(7 * 24 * time.Hour)

	accessClaims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Provider: user.Provider,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}

	refreshClaims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Provider: user.Provider,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	accessTokenString, err := accessToken.SignedString(p.jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshTokenString, err := refreshToken.SignedString(p.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
	}, nil
}

func (p *LocalProvider) CreateUser(username, email, password string, role Role) error {
	if _, exists := p.users[username]; exists {
		return fmt.Errorf("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	userID := uuid.New()
	p.users[username] = &User{
		ID:       userID,
		Username: username,
		Email:    email,
		Role:     role,
		Provider: "local",
		Active:   true,
	}
	p.passwords[username] = string(hashedPassword)

	return nil
}

type OAuthProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	jwtSecret    []byte
}

func NewOAuthProvider(clientID, clientSecret, redirectURL, jwtSecret string) *OAuthProvider {
	return &OAuthProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		jwtSecret:    []byte(jwtSecret),
	}
}

func (p *OAuthProvider) Authenticate(ctx context.Context, creds Credentials) (*User, error) {
	// OAuth implementation would validate token with provider
	// For now, return mock user
	return &User{
		ID:       uuid.New(),
		Username: "oauth_user",
		Email:    "user@oauth.com",
		Role:     RoleDeveloper,
		Provider: "oauth",
		Active:   true,
		LastLogin: time.Now(),
	}, nil
}

func (p *OAuthProvider) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return p.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (p *OAuthProvider) RefreshToken(refreshToken string) (*TokenPair, error) {
	// OAuth refresh implementation
	return nil, fmt.Errorf("not implemented")
}

type LDAPProvider struct {
	server    string
	baseDN    string
	jwtSecret []byte
}

func NewLDAPProvider(server, baseDN, jwtSecret string) *LDAPProvider {
	return &LDAPProvider{
		server:    server,
		baseDN:    baseDN,
		jwtSecret: []byte(jwtSecret),
	}
}

func (p *LDAPProvider) Authenticate(ctx context.Context, creds Credentials) (*User, error) {
	// LDAP implementation would connect to LDAP server
	// For now, return mock user
	return &User{
		ID:       uuid.New(),
		Username: creds.Username,
		Email:    creds.Username + "@ldap.com",
		Role:     RoleResearcher,
		Provider: "ldap",
		Active:   true,
		LastLogin: time.Now(),
	}, nil
}

func (p *LDAPProvider) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return p.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (p *LDAPProvider) RefreshToken(refreshToken string) (*TokenPair, error) {
	// LDAP refresh implementation
	return nil, fmt.Errorf("not implemented")
}

type SessionManager struct {
	sessions map[string]*Session
	cleanup  chan struct{}
}

type Session struct {
	ID        string    `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		cleanup:  make(chan struct{}),
	}
	go sm.cleanupExpiredSessions()
	return sm
}

func (sm *SessionManager) CreateSession(userID uuid.UUID, ipAddress, userAgent string) *Session {
	sessionID := generateSecureToken()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
	
	sm.sessions[sessionID] = session
	return session
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	session, exists := sm.sessions[sessionID]
	if !exists || time.Now().After(session.ExpiresAt) {
		delete(sm.sessions, sessionID)
		return nil, false
	}
	return session, true
}

func (sm *SessionManager) DeleteSession(sessionID string) {
	delete(sm.sessions, sessionID)
}

func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			for id, session := range sm.sessions {
				if now.After(session.ExpiresAt) {
					delete(sm.sessions, id)
				}
			}
		case <-sm.cleanup:
			return
		}
	}
}

func generateSecureToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}