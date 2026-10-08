package computeruse

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResultMetadataPreservesBoundIdentityWithoutPrivateFields(t *testing.T) {
	output := `{"launch_receipt":{"target_id":"fixture","bundle_id":"fixture.app","outcome":"executed","window":{"id":"42","title":"PRIVATE DOCUMENT","owner_pid":7,"bundle_id":"fixture.app","frame":{"x":1,"y":2,"width":3,"height":4}}},"observation":{"window_id":"42","active_window":{"id":"42","title":"PRIVATE DOCUMENT","owner_pid":7,"bundle_id":"fixture.app"}},"private":"SECRET"}`
	meta := ResultMetadata(output)
	if meta["window_id"] != "42" || meta["bundle_id"] != "fixture.app" || meta["launch_receipt"] == nil {
		t.Fatalf("missing identity: %#v", meta)
	}
	data, _ := json.Marshal(meta)
	if strings.Contains(string(data), "PRIVATE DOCUMENT") || strings.Contains(string(data), "SECRET") {
		t.Fatalf("private data leaked: %s", data)
	}
}
