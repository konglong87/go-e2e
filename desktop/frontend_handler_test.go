package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDesktopPageNavigationUsesPackagedFrontend(t *testing.T) {
	const index = "<!doctype html><div id=\"root\"></div>"
	assets := fstest.MapFS{desktopIndexPath: {Data: []byte(index)}}
	handler := newDesktopHandler(assets, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("page navigation unexpectedly reached runtime proxy: %s", r.URL.Path)
		w.WriteHeader(http.StatusBadGateway)
	}))
	for _, path := range []string{"/webui", "/webui/", "/webui/agent?token=test-token", "/webui/agent/"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(method+" "+path, func(t *testing.T) {
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(method, path, nil))
				if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "text/html") {
					t.Fatalf("unexpected response: %d %v", response.Code, response.Header())
				}
				if method == http.MethodGet && response.Body.String() != index {
					t.Fatalf("expected packaged index, got %q", response.Body.String())
				}
				if method == http.MethodHead && response.Body.Len() != 0 {
					t.Fatal("HEAD must not return a body")
				}
			})
		}
	}
}

func TestDesktopFrontendFallbackPreservesRuntimeRequests(t *testing.T) {
	for _, path := range []string{"/tenant/prompt-templates", "/api/tenant/prompt-templates", "/health", "/trace", "/webui/assets/missing.js", "/webui/unknown"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path+"?search=review", nil)
			request.Header.Set("Authorization", "Bearer test-token")
			request.Header.Set("X-Tenant-Key", "desktop-tenant")
			request.Header.Set("X-User-ID", "desktop-user")
			called := false
			handler := newDesktopHandler(fstest.MapFS{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r != request {
					t.Fatal("proxy must receive the original request including identity and query")
				}
				w.WriteHeader(http.StatusUnauthorized)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if !called || response.Code != http.StatusUnauthorized {
				t.Fatalf("API response changed: called=%v status=%d", called, response.Code)
			}
		})
	}
}

func TestDesktopPagePostDoesNotReturnFrontend(t *testing.T) {
	handler := newDesktopHandler(fstest.MapFS{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/webui/agent", strings.NewReader("{}")))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST was intercepted: %d", response.Code)
	}
}
