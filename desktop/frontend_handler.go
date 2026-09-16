package main

import (
	"io/fs"
	"net/http"
)

const desktopIndexPath = "frontend/dist/index.html"

// Page navigation must use the packaged frontend, independent of server cwd.
// All API and non-page requests still pass through the existing runtime proxy.
func newDesktopHandler(assets fs.FS, api http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			switch r.URL.Path {
			case "/webui", "/webui/", "/webui/agent", "/webui/agent/":
				http.ServeFileFS(w, r, assets, desktopIndexPath)
				return
			}
		}
		api.ServeHTTP(w, r)
	})
}
