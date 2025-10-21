package assets

import (
	"embed"
	"net/http"
)

//go:embed assets/*
var AdminAssets embed.FS

var EmbeddedFS = http.FileServer(http.FS(AdminAssets))
var AssetFS = http.FileServer(http.Dir("pkg/admin/assets/assets"))
