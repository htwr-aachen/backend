package assets

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets/*
var AdminAssets embed.FS

var EmbeddedFS http.Handler

func Init() {
	// Strip the "assets/" prefix from the embedded FS
	sub, _ := fs.Sub(AdminAssets, "assets")
	EmbeddedFS = http.FileServer(http.FS(sub))
}

var AssetFS = http.FileServer(http.Dir("pkg/admin/assets/assets"))
