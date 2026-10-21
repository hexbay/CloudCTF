//go:build embed

package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded frontend build output (the contents of dist/).
// The second return value reports whether an embedded build is available.
func DistFS() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
