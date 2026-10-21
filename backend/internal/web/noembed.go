//go:build !embed

package web

import "io/fs"

// DistFS returns no embedded filesystem in non-embed builds. The server then
// falls back to serving the frontend from cfg.FrontendDir on disk (useful for
// local development).
func DistFS() (fs.FS, bool) {
	return nil, false
}
