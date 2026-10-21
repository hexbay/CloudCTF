package httpserver

import (
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"cloudctf/backend/internal/config"
	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/web"
)

// NewRouter builds a Server from deps and returns its HTTP handler.
func NewRouter(deps Dependencies) http.Handler {
	return NewServer(deps).Handler()
}

// Handler assembles the HTTP routing tree and middleware chain for the server.
func (srv *Server) Handler() http.Handler {
	cfg := srv.config
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	mountStatic(mux, "/static/", cfg.StaticDir)
	mountStatic(mux, "/static/uploads/", cfg.UploadDir)

	srv.registerAPIRoutes(mux)

	// Serve the bundled single-page frontend (admin + player) as a catch-all.
	// If no frontend is available, fall back to an API welcome message at root.
	if !mountFrontend(mux, cfg) {
		mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			response.JSON(w, http.StatusOK, map[string]any{
				"message": "Welcome to CloudCTF API",
				"service": cfg.AppName,
			})
		})
	}

	handler := recoverer(srv.logger, mux)
	handler = srv.adminAuth(handler)
	handler = requestLogger(srv.logger, handler)
	handler = sensitiveRateLimiter(handler)
	handler = cors(cfg.AllowedOrigins, handler)
	handler = securityHeaders(handler)
	handler = requestProtection(handler)
	return handler
}

// mountFrontend serves the unified SPA build at the root path. It prefers the
// build output embedded into the binary (go build -tags embed); otherwise it
// falls back to cfg.FrontendDir on disk. Unknown paths fall back to index.html
// so client-side routes such as /admin/... resolve correctly. It returns false
// when no frontend build can be located.
func mountFrontend(mux *http.ServeMux, cfg config.Config) bool {
	var fsys fs.FS
	if embedded, ok := web.DistFS(); ok {
		fsys = embedded
	} else if dir := strings.TrimSpace(cfg.FrontendDir); dir != "" {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return false
		}
		fsys = os.DirFS(dir)
	} else {
		return false
	}

	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return false
	}

	fileServer := http.FileServer(http.FS(fsys))
	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upath := strings.TrimPrefix(r.URL.Path, "/")
		if upath == "" {
			serveIndex(w, r)
			return
		}
		if f, err := fsys.Open(upath); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// Client-side route: serve the SPA shell.
		serveIndex(w, r)
	})
	return true
}

func mountStatic(mux *http.ServeMux, prefix, dir string) {
	if _, err := os.Stat(dir); err != nil {
		return
	}
	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.Dir(dir))))
}

func apiRouteFallback(module string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusNotFound, map[string]any{
			"detail": "route not found",
			"module": module,
			"path":   r.URL.Path,
		})
	}
}
