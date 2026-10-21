package httpserver

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/models"
	"cloudctf/backend/internal/services"
)

// ================================================
// Admin auth handlers
// ================================================

func (srv *Server) adminLoginHandler(w http.ResponseWriter, r *http.Request) {
	username, password, err := readAdminCredentials(r)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	result, err := srv.getAuthService().LoginAdmin(username, password)
	if err != nil {
		srv.auditLoginFailed(username, "admin", r)
		writeAuthError(w, err)
		return
	}
	srv.auditLoginSuccess(result.User.ID, result.User.Username, "admin", r)
	response.JSON(w, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"token_type":    "bearer",
		"expires_at":    result.ExpiresAt,
		"user":          adminUserResponse(result.User),
	})
}

func (srv *Server) adminMeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	response.JSON(w, http.StatusOK, adminUserResponse(user))
}

func (srv *Server) adminRefreshHandler(w http.ResponseWriter, r *http.Request) {
	payload, _ := readOptionalJSONMap(r)
	token := stringifyID(payload["refresh_token"])
	user, accessToken, refreshToken, expiresAt, ok := srv.rotateRefreshToken(token)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
		"expires_at":    expiresAt,
		"user":          adminUserResponse(user),
	})
}

func (srv *Server) adminLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if user, ok := srv.currentUserFromRequest(r); ok {
		_ = srv.revokeAllUserRefreshTokens(user.ID)
	}
	if payload, ok := readOptionalJSONMap(r); ok {
		_ = srv.revokeRefreshToken(stringifyID(payload["refresh_token"]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"msg": "ok"})
}

func readAdminCredentials(r *http.Request) (string, string, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") || strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseForm(); err != nil {
			return "", "", err
		}
		return r.FormValue("username"), r.FormValue("password"), nil
	}

	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return "", "", err
	}
	return payload.Username, payload.Password, nil
}

func adminUserResponse(user userModel) adminUserMe {
	return adminUserMe{
		ID:                 int(user.ID),
		Username:           user.Username,
		Email:              user.Email,
		Nickname:           user.FullName,
		Role:               user.Role.Name,
		IsSuperuser:        user.IsSuperuser,
		MustChangePassword: user.MustChangePassword,
		Permissions:        user.Role.PermissionList(),
	}
}

type adminUserMe struct {
	ID                 int      `json:"id"`
	Username           string   `json:"username"`
	Email              string   `json:"email"`
	Nickname           string   `json:"nickname"`
	Role               string   `json:"role"`
	IsSuperuser        bool     `json:"is_superuser"`
	MustChangePassword bool     `json:"must_change_password"`
	Permissions        []string `json:"permissions"`
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrAuthBadRequest):
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": "username and password are required"})
	case errors.Is(err, services.ErrAuthUnauthorized):
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Incorrect username or password"})
	case errors.Is(err, services.ErrAuthInactiveUser):
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "Inactive user"})
	case errors.Is(err, services.ErrAuthForbidden):
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": "Not enough permissions"})
	default:
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to issue auth tokens"})
	}
}

// ================================================
// Identity lookup helpers
// ================================================

func (srv *Server) findUserByLogin(login string) (userModel, bool) {
	service := srv.getUserService()
	if service == nil {
		return userModel{}, false
	}
	user, err := service.FindByLogin(login)
	if err != nil {
		return userModel{}, false
	}
	return *user, true
}

func (srv *Server) findUserByID(id uint) (userModel, bool) {
	service := srv.getUserService()
	if service == nil {
		return userModel{}, false
	}
	user, err := service.GetByID(id)
	if err != nil {
		return userModel{}, false
	}
	return *user, true
}

// ================================================
// Token issuance / validation
// ================================================

const (
	accessTokenTTL  = 24 * time.Hour
	refreshTokenTTL = 7 * 24 * time.Hour
)

type accessTokenClaims struct {
	UserID    uint   `json:"uid"`
	TokenType string `json:"typ"`
	ExpiresAt int64  `json:"exp"`
}

type streamTicket struct {
	UserID    uint
	Path      string
	ExpiresAt time.Time
}

func (srv *Server) streamTicketHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	path := strings.TrimSpace(fmt.Sprint(payload["path"]))
	if !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/api-admin/") {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "valid stream path required"})
		return
	}
	ticket, err := randomToken(32)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to issue stream ticket"})
		return
	}
	expiresAt := time.Now().Add(30 * time.Second)
	srv.streamTicketsMu.Lock()
	for key, item := range srv.streamTickets {
		if time.Now().After(item.ExpiresAt) {
			delete(srv.streamTickets, key)
		}
	}
	srv.streamTickets[ticket] = streamTicket{UserID: user.ID, Path: path, ExpiresAt: expiresAt}
	srv.streamTicketsMu.Unlock()
	response.JSON(w, http.StatusOK, map[string]any{"ticket": ticket, "expires_at": expiresAt})
}

func (srv *Server) consumeStreamTicket(ticket, path string) (uint, bool) {
	srv.streamTicketsMu.Lock()
	defer srv.streamTicketsMu.Unlock()
	item, ok := srv.streamTickets[ticket]
	delete(srv.streamTickets, ticket)
	return item.UserID, ok && item.Path == path && time.Now().Before(item.ExpiresAt)
}

func (srv *Server) issueAccessToken(userID uint, ttl time.Duration) (string, time.Time, error) {
	expiresAt := time.Now().Add(ttl)
	payload, err := json.Marshal(accessTokenClaims{
		UserID:    userID,
		TokenType: "access",
		ExpiresAt: expiresAt.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := srv.signTokenPayload(encodedPayload)
	return encodedPayload + "." + signature, expiresAt, nil
}

func (srv *Server) parseAccessToken(token string) (uint, bool) {
	payloadPart, signaturePart, ok := strings.Cut(strings.TrimSpace(token), ".")
	if !ok || payloadPart == "" || signaturePart == "" {
		return 0, false
	}
	expectedSignature := srv.signTokenPayload(payloadPart)
	if !hmac.Equal([]byte(expectedSignature), []byte(signaturePart)) {
		return 0, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return 0, false
	}
	var claims accessTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, false
	}
	if claims.TokenType != "access" || claims.UserID == 0 || time.Now().Unix() > claims.ExpiresAt {
		return 0, false
	}
	return claims.UserID, true
}

func (srv *Server) issueRefreshToken(userID uint, ttl time.Duration) (string, error) {
	db := srv.db
	if db == nil {
		return "", fmt.Errorf("database is not configured")
	}
	rawToken, err := randomToken(32)
	if err != nil {
		return "", err
	}
	record := models.RefreshToken{
		UserID:    userID,
		TokenHash: hashRefreshToken(rawToken),
		ExpiresAt: time.Now().Add(ttl),
	}
	if err := db.Create(&record).Error; err != nil {
		return "", err
	}
	return rawToken, nil
}

func (srv *Server) rotateRefreshToken(token string) (userModel, string, string, time.Time, bool) {
	record, ok := srv.findActiveRefreshToken(token)
	if !ok {
		return userModel{}, "", "", time.Time{}, false
	}
	user, ok := srv.findUserByID(record.UserID)
	if !ok || !user.IsActive {
		_ = srv.revokeRefreshToken(token)
		return userModel{}, "", "", time.Time{}, false
	}
	now := time.Now()
	record.RevokedAt = &now
	if err := srv.db.Save(&record).Error; err != nil {
		return userModel{}, "", "", time.Time{}, false
	}
	accessToken, expiresAt, err := srv.issueAccessToken(user.ID, accessTokenTTL)
	if err != nil {
		return userModel{}, "", "", time.Time{}, false
	}
	refreshToken, err := srv.issueRefreshToken(user.ID, refreshTokenTTL)
	if err != nil {
		return userModel{}, "", "", time.Time{}, false
	}
	return user, accessToken, refreshToken, expiresAt, true
}

func (srv *Server) issueAuthTokens(userID uint) (string, string, time.Time, error) {
	accessToken, expiresAt, err := srv.issueAccessToken(userID, accessTokenTTL)
	if err != nil {
		return "", "", time.Time{}, err
	}
	refreshToken, err := srv.issueRefreshToken(userID, refreshTokenTTL)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return accessToken, refreshToken, expiresAt, nil
}

func (srv *Server) revokeRefreshToken(token string) error {
	db := srv.db
	if db == nil || strings.TrimSpace(token) == "" {
		return nil
	}
	hash := hashRefreshToken(token)
	now := time.Now()
	return db.Model(&models.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", &now).Error
}

func (srv *Server) revokeAllUserRefreshTokens(userID uint) error {
	db := srv.db
	if db == nil || userID == 0 {
		return nil
	}
	now := time.Now()
	return db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", &now).Error
}

func (srv *Server) currentUserFromRequest(r *http.Request) (userModel, bool) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token != "" {
		userID, ok := srv.parseAccessToken(token)
		if !ok {
			return userModel{}, false
		}
		return srv.findUserByID(userID)
	}
	userID, ok := srv.consumeStreamTicket(strings.TrimSpace(r.URL.Query().Get("stream_ticket")), r.URL.Path)
	if !ok {
		return userModel{}, false
	}
	return srv.findUserByID(userID)
}

func (srv *Server) findActiveRefreshToken(token string) (models.RefreshToken, bool) {
	db := srv.db
	if db == nil || strings.TrimSpace(token) == "" {
		return models.RefreshToken{}, false
	}
	var record models.RefreshToken
	if err := db.Where("token_hash = ? AND revoked_at IS NULL", hashRefreshToken(token)).First(&record).Error; err != nil {
		return models.RefreshToken{}, false
	}
	if time.Now().After(record.ExpiresAt) {
		_ = srv.revokeRefreshToken(token)
		return models.RefreshToken{}, false
	}
	return record, true
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (srv *Server) signTokenPayload(payload string) string {
	secret := srv.currentConfig().JWTSecret
	if strings.TrimSpace(secret) == "" {
		secret = "change-me"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ================================================
// Token issuer adapter (implements services TokenIssuer)
// ================================================

type httpTokenIssuer struct {
	srv *Server
}

func (t httpTokenIssuer) IssueAuthTokens(userID uint) (string, string, time.Time, error) {
	return t.srv.issueAuthTokens(userID)
}
