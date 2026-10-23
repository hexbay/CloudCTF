package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloudctf/backend/internal/httpserver/response"
)

const (
	defaultRequestBodyLimit      int64 = 2 << 20
	uploadRequestBodyLimit       int64 = 64 << 20
	updateUploadRequestBodyLimit int64 = (512 << 20) + (1 << 20)
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,128}$`)

func requestProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if !requestIDPattern.MatchString(requestID) {
			var raw [16]byte
			if _, err := rand.Read(raw[:]); err == nil {
				requestID = hex.EncodeToString(raw[:])
			} else {
				requestID = strconv.FormatInt(time.Now().UnixNano(), 36)
			}
		}
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)

		if r.Body != nil && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
			limit := requestBodyLimit(r.URL.Path)
			if r.ContentLength > limit {
				response.JSON(w, http.StatusRequestEntityTooLarge, map[string]any{"detail": "request body too large"})
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func requestBodyLimit(path string) int64 {
	if path == "/api-admin/settings/update/upload" {
		return updateUploadRequestBodyLimit
	}
	for _, marker := range []string{"/upload", "/import", "/files", "/images/load"} {
		if strings.Contains(path, marker) {
			return uploadRequestBodyLimit
		}
	}
	return defaultRequestBodyLimit
}

type rateLimitEntry struct {
	count int
	reset time.Time
}

const sensitiveRateLimitMaxEntries = 10_000

func sensitiveRateLimiter(next http.Handler) http.Handler {
	return newSensitiveRateLimiter(next, sensitiveRateLimitMaxEntries)
}

func newSensitiveRateLimiter(next http.Handler, maxEntries int) http.Handler {
	var mu sync.Mutex
	entries := map[string]rateLimitEntry{}
	lastCleanup := time.Now()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := sensitiveRouteLimit(r)
		if limit == 0 {
			next.ServeHTTP(w, r)
			return
		}
		now := time.Now()
		key := clientIP(r) + "|" + r.Method + "|" + r.URL.Path
		mu.Lock()
		if now.Sub(lastCleanup) >= time.Minute {
			pruneExpiredRateLimitEntries(entries, now)
			lastCleanup = now
		}
		entry, exists := entries[key]
		if !exists && len(entries) >= maxEntries {
			mu.Unlock()
			w.Header().Set("Retry-After", "60")
			response.JSON(w, http.StatusTooManyRequests, map[string]any{"detail": "too many requests"})
			return
		}
		if entry.reset.IsZero() || now.After(entry.reset) {
			entry = rateLimitEntry{reset: now.Add(time.Minute)}
		}
		entry.count++
		entries[key] = entry
		mu.Unlock()
		if entry.count > limit {
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int(time.Until(entry.reset).Seconds()))))
			response.JSON(w, http.StatusTooManyRequests, map[string]any{"detail": "too many requests"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func pruneExpiredRateLimitEntries(entries map[string]rateLimitEntry, now time.Time) {
	for key, entry := range entries {
		if !entry.reset.After(now) {
			delete(entries, key)
		}
	}
}

func sensitiveRouteLimit(r *http.Request) int {
	path := r.URL.Path
	if r.Method == http.MethodPost && (path == "/api/login" || path == "/api/register" || path == "/api-admin/auth/login") {
		return 10
	}
	if r.Method == http.MethodPost && strings.Contains(path, "reset-password") {
		return 10
	}
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/submit") {
		return 60
	}
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/start") {
		return 20
	}
	return 0
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self' ws: wss:; font-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func cors(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isOriginAllowed(origin, allowedOrigins) {
			if origin == "" || contains(allowedOrigins, "*") {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request completed",
			"request_id", r.Header.Get("X-Request-ID"),
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}

func recoverer(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "error", recovered, "stack", string(debug.Stack()))
				response.JSON(w, http.StatusInternalServerError, map[string]any{
					"detail": "Internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (srv *Server) adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api-admin/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodOptions || isPublicAdminRoute(r) {
			next.ServeHTTP(w, r)
			return
		}

		user, ok := srv.currentUserFromRequest(r)
		if !ok {
			response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
			return
		}
		if user.Role.Name == rolePlayer && !user.IsSuperuser {
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": "admin permissions required"})
			return
		}
		if permission := adminRoutePermission(r.URL.Path); permission != "" && !userHasAdminPermission(user, permission) {
			response.JSON(w, http.StatusForbidden, map[string]any{
				"detail":     "permission required",
				"permission": permission,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func adminRoutePermission(path string) string {
	relative := strings.TrimPrefix(path, "/api-admin/")
	segment, _, _ := strings.Cut(relative, "/")
	switch segment {
	case "", "auth", "dashboard":
		return ""
	case "users", "user-management", "referral-domain-stats":
		return "admin:users"
	case "admins", "roles", "resource-permissions":
		return "admin:roles"
	case "competitions":
		return "admin:competitions"
	case "challenges", "flags", "upload":
		return "admin:challenges"
	case "theory":
		return "admin:theory"
	case "exam":
		return "admin:exam"
	case "teams":
		return "admin:teams"
	case "submissions":
		return "admin:submissions"
	case "scoreboard":
		return "admin:scoreboard"
	case "notifications":
		return "admin:notifications"
	case "announcements":
		return "admin:announcements"
	case "audit":
		return "admin:audit"
	case "virtualization":
		return "admin:virtualization"
	case "vuln-labs":
		return "admin:vuln_labs"
	case "resource-repositories":
		return "admin:resource_repositories"
	case "settings":
		return "admin:settings"
	case "writeups":
		return "admin:writeup"
	default:
		return "admin:" + segment
	}
}

func userHasAdminPermission(user userModel, required string) bool {
	if user.IsSuperuser {
		return true
	}
	for _, permission := range user.Role.PermissionList() {
		permission = strings.TrimSpace(permission)
		if permission == "*" || permission == "admin:*" || permission == required {
			return true
		}
		if prefix, ok := strings.CutSuffix(permission, "*"); ok && strings.HasPrefix(required, prefix) {
			return true
		}
	}
	return false
}

func isPublicAdminRoute(r *http.Request) bool {
	switch r.URL.Path {
	case "/api-admin/auth/login":
		return r.Method == http.MethodPost
	default:
		return false
	}
}

func isOriginAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return true
	}
	if contains(allowed, "*") {
		return true
	}
	return contains(allowed, origin)
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), want) {
			return true
		}
	}
	return false
}
