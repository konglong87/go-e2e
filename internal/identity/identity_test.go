package identity

import "testing"

func TestFromSettingsAppliesConfiguredNames(t *testing.T) {
	id := FromSettings(Settings{
		ProductName:      "agentx",
		ProductKey:       "agentx",
		ConfigDirName:    ".agentx",
		GuidanceFilename: "agentx.md",
	})
	if id.ProductName != "agentx" || id.ProductKey != "agentx" || id.ConfigDirName != ".agentx" || id.GuidanceFilename != "agentx.md" {
		t.Fatalf("identity = %+v", id)
	}
	if id.LegacyGuidanceFile != "CLAUDE.md" || id.WorkflowFallbackFile != "AGENTS.md" {
		t.Fatalf("legacy compatibility names changed: %+v", id)
	}
}

func TestFromSettingsFallsBackOnUnsafeNames(t *testing.T) {
	id := FromSettings(Settings{GuidanceFilename: "../bad.md"})
	if id.GuidanceFilename != "golang-cc.md" || id.ConfigDirName != ".golang-cc" {
		t.Fatalf("identity did not fall back to defaults: %+v", id)
	}
}
