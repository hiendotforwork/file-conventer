package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed templates/index.html static
var files embed.FS

func Index() []byte {
	data, _ := files.ReadFile("templates/index.html")
	return data
}

func Static() http.Handler {
	static, _ := fs.Sub(files, "static")
	return http.FileServer(http.FS(static))
}
