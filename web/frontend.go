package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var frontendFS embed.FS

// GetFS returns the frontend dist filesystem
func GetFS() fs.FS {
	sub, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return nil
	}
	return sub
}
