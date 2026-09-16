package server

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/konglong87/go-e2e/internal/config"
)

// settingsSecretSentinel is the placeholder shown to the WebUI in place of a
// credential value. When a PUT sends this sentinel back unchanged, the stored
// value is restored, so editing unrelated fields never wipes a secret.
const settingsSecretSentinel = "••••••"

// Serializes API read-compare-write operations across handlers in this process.
// External editors and other server processes must still coordinate separately.
var globalSettingsMu sync.Mutex

// GlobalSettingsResponse is the payload returned by GET /runtime/settings.
type GlobalSettingsResponse struct {
	Path     string         `json:"path"`
	Exists   bool           `json:"exists"`
	Doc      map[string]any `json:"doc"`
	Masked   []string       `json:"masked"`
	Revision string         `json:"revision"`
}

// GlobalSettingsSaveResponse is the payload returned by PUT /runtime/settings.
type GlobalSettingsSaveResponse struct {
	Path            string `json:"path"`
	Saved           bool   `json:"saved"`
	Revision        string `json:"revision"`
	RequiresRestart bool   `json:"requires_restart"`
}

func registerSettingsRoutes(router *gin.Engine, opts Options) {
	router.Any("/runtime/settings", runtimeSettingsGin(opts))
	router.POST("/runtime/settings/validate", settingsValidateGin(opts))
	router.GET("/runtime/settings/effective", settingsEffectiveGin(opts))
	router.POST("/runtime/settings/test-provider", settingsTestProviderGin(opts))
}

func runtimeSettingsGin(opts Options) gin.HandlerFunc {
	return func(c *gin.Context) {
		logService(c.Request.Context(), "runtime.settings", "server.runtimeSettingsGin", "read or write global settings")
		if !authorize(c.Writer, c.Request, opts.AuthToken) {
			return
		}
		switch c.Request.Method {
		case http.MethodGet:
			handleGetGlobalSettings(c.Writer)
		case http.MethodPut, http.MethodPost:
			handlePutGlobalSettings(c.Writer, c.Request)
		default:
			http.Error(c.Writer, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleGetGlobalSettings(w http.ResponseWriter) {
	data, path, ok, err := config.ReadGlobalSettingsRaw()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := GlobalSettingsResponse{Path: path, Exists: ok, Doc: map[string]any{}, Masked: []string{}, Revision: settingsRevision(data, ok)}
	if ok {
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil || doc == nil {
			http.Error(w, "global settings.json must contain a JSON object", http.StatusInternalServerError)
			return
		}
		masked := []string{}
		maskSecrets(doc, "", &masked)
		sort.Strings(masked)
		resp.Doc = doc
		resp.Masked = masked
	}
	w.Header().Set("ETag", `"`+resp.Revision+`"`)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, resp)
}

func handlePutGlobalSettings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil || doc == nil {
		http.Error(w, "request body must be a JSON settings object", http.StatusBadRequest)
		return
	}
	// Type-check against the known Settings schema. Unknown keys are ignored
	// here (so forward-compatible fields survive) but any type mismatch on a
	// known field is rejected before we touch the file on disk.
	if issues := validateSettingsDocument(body); len(issues) > 0 {
		writeJSONStatus(w, http.StatusBadRequest, SettingsValidationResponse{Valid: false, Issues: issues})
		return
	}
	globalSettingsMu.Lock()
	defer globalSettingsMu.Unlock()
	// Restore any masked placeholders the client echoed back to their stored
	// values so a round-trip through the WebUI never blanks a credential.
	oldDoc := map[string]any{}
	raw, _, exists, readErr := config.ReadGlobalSettingsRaw()
	if readErr != nil {
		http.Error(w, "cannot read current global settings", http.StatusInternalServerError)
		return
	}
	revision := settingsRevision(raw, exists)
	if match := strings.TrimSpace(r.Header.Get("If-Match")); match != "" && match != revision && match != `"`+revision+`"` {
		writeJSONStatus(w, http.StatusConflict, map[string]any{"error": "settings_conflict", "revision": revision})
		return
	}
	if exists {
		if err := json.Unmarshal(raw, &oldDoc); err != nil || oldDoc == nil {
			http.Error(w, "current global settings must contain a valid JSON object", http.StatusInternalServerError)
			return
		}
	}
	if err := restoreSecrets(doc, oldDoc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out = append(out, '\n')
	path, err := config.WriteGlobalSettingsRaw(out)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logService(r.Context(), "runtime.settings.saved", "server.handlePutGlobalSettings", "global settings saved; active runs unchanged")
	revision = settingsRevision(out, true)
	w.Header().Set("ETag", `"`+revision+`"`)
	writeJSON(w, GlobalSettingsSaveResponse{Path: path, Saved: true, Revision: revision, RequiresRestart: true})
}

func settingsRevision(data []byte, exists bool) string {
	return fmt.Sprintf("%x", sha256.Sum256(append([]byte(fmt.Sprint(exists)+":"), data...)))
}

// isSecretKey reports whether an object key names a credential-like value that
// should be masked before leaving the server.
func isSecretKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, needle := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "PASSWD", "AUTHORIZATION", "COOKIE", "CREDENTIAL"} {
		if strings.Contains(upper, needle) {
			return true
		}
	}
	return false
}

// maskSecrets walks a decoded JSON document in place, replacing the value of
// any object key that looks like a credential with settingsSecretSentinel and
// recording the dotted path of each masked field in masked.
func maskSecrets(node any, path string, masked *[]string) {
	switch value := node.(type) {
	case map[string]any:
		for key, child := range value {
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			if str, ok := child.(string); ok && strings.TrimSpace(str) != "" && settingsSecretValue(key, path, str) {
				value[key] = settingsSecretSentinel
				*masked = append(*masked, childPath)
				continue
			}
			maskSecrets(child, childPath, masked)
		}
	case []any:
		for i, child := range value {
			maskSecrets(child, fmt.Sprintf("%s.%d", path, i), masked)
		}
	}
}

// restoreSecrets walks newNode in place, replacing every sentinel placeholder
// with the real value at the matching position in oldNode. A sentinel with no
// stored counterpart is a hard error, so a placeholder is never written to disk
// as if it were a real secret.
func restoreSecrets(newNode, oldNode any) error {
	return restoreSettingsSecrets(newNode, oldNode, "")
}

func restoreSettingsSecrets(newNode, oldNode any, path string) error {
	switch value := newNode.(type) {
	case map[string]any:
		oldMap, _ := oldNode.(map[string]any)
		for key, child := range value {
			if str, ok := child.(string); ok && str == settingsSecretSentinel {
				prev, ok := oldMap[key].(string)
				if !ok || strings.TrimSpace(prev) == "" {
					return errors.New("cannot save masked placeholder for \"" + key + "\": no stored value to restore, enter a real value")
				}
				value[key] = prev
				continue
			}
			var oldChild any
			if oldMap != nil {
				oldChild = oldMap[key]
			}
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			if err := restoreSettingsSecrets(child, oldChild, childPath); err != nil {
				return err
			}
		}
	case []any:
		oldSlice, _ := oldNode.([]any)
		for i, child := range value {
			var oldChild any
			item, _ := child.(map[string]any)
			name, _ := item["name"].(string)
			name = strings.TrimSpace(name)
			if name != "" {
				// Named provider credentials follow identity when rows move. A rename
				// cannot safely inherit a masked value and requires re-entry.
				for _, candidate := range oldSlice {
					if previous, ok := candidate.(map[string]any); ok {
						previousName, _ := previous["name"].(string)
						if strings.TrimSpace(previousName) != name {
							continue
						}
						if oldChild != nil {
							oldChild = nil
							break
						}
						oldChild = previous
					}
				}
			} else if path == "fallback.providers" {
				// Unnamed legacy routes have no stable name. Only an unambiguous
				// endpoint/model/protocol identity may retain a masked credential.
				oldChild = matchingUnnamedSettingsProvider(item, oldSlice)
			} else if oldSlice != nil && i < len(oldSlice) {
				oldChild = oldSlice[i]
			}
			if err := restoreSettingsSecrets(child, oldChild, fmt.Sprintf("%s.%d", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func matchingUnnamedSettingsProvider(item map[string]any, candidates []any) any {
	var matched map[string]any
	for _, candidate := range candidates {
		previous, ok := candidate.(map[string]any)
		if !ok {
			continue
		}
		if name, _ := previous["name"].(string); strings.TrimSpace(name) != "" {
			continue
		}
		matches := true
		for _, field := range []string{"type", "protocol", "model", "baseURL"} {
			currentValue, _ := item[field].(string)
			previousValue, _ := previous[field].(string)
			if settingsSecretValue(field, "", previousValue) {
				previousValue = settingsSecretSentinel
			}
			if currentValue != previousValue {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		if matched != nil {
			return nil
		}
		matched = previous
	}
	return matched
}
