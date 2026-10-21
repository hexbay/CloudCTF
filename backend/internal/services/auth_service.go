package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAuthBadRequest   = errors.New("username and password are required")
	ErrAuthUnauthorized = errors.New("incorrect username or password")
	ErrAuthInactiveUser = errors.New("inactive user")
	ErrAuthForbidden    = errors.New("not enough permissions")
)

type TokenIssuer interface {
	IssueAuthTokens(userID uint) (accessToken string, refreshToken string, expiresAt time.Time, err error)
}

type AuthService struct {
	users  *UserService
	tokens TokenIssuer
}

type AuthenticatedUser struct {
	User         models.User
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func NewAuthService(users *UserService, tokens TokenIssuer) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

func (s *AuthService) LoginAdmin(username, password string) (AuthenticatedUser, error) {
	return s.login(username, password, true)
}

func (s *AuthService) LoginPlayer(username, password string) (AuthenticatedUser, error) {
	return s.login(username, password, false)
}

func (s *AuthService) login(username, password string, requireAdmin bool) (AuthenticatedUser, error) {
	if strings.TrimSpace(username) == "" || password == "" {
		return AuthenticatedUser{}, ErrAuthBadRequest
	}
	user, err := s.users.FindByLogin(username)
	if err != nil || !VerifyPassword(password, user.HashedPassword) {
		return AuthenticatedUser{}, ErrAuthUnauthorized
	}
	if !user.IsActive {
		return AuthenticatedUser{}, ErrAuthInactiveUser
	}
	if requireAdmin && user.Role.Name == "player" && !user.IsSuperuser {
		return AuthenticatedUser{}, ErrAuthForbidden
	}
	accessToken, refreshToken, expiresAt, err := s.tokens.IssueAuthTokens(user.ID)
	if err != nil {
		return AuthenticatedUser{}, err
	}
	return AuthenticatedUser{
		User:         *user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func VerifyPassword(password, hashed string) bool {
	if strings.HasPrefix(hashed, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
	}
	legacy := sha256.Sum256([]byte(password))
	return hex.EncodeToString(legacy[:]) == hashed
}
