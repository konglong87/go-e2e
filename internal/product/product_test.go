package product

import (
	"os"
	"testing"
)

func TestCanonicalInternalIdentifiers(t *testing.T) {
	wants := map[string]string{
		"transcript v1":      TranscriptSchemaV1,
		"transcript v2":      TranscriptSchemaV2,
		"tenant quota redis": TenantQuotaRedisPrefix,
		"mobile usage redis": MobileUsageRedisPrefix,
	}
	expected := map[string]string{
		"transcript v1":      "golang-cc.transcript.v1",
		"transcript v2":      "golang-cc.transcript.v2",
		"tenant quota redis": "golang-cc:tenant_quota",
		"mobile usage redis": "golang-cc:mobile_usage",
	}
	for name, got := range wants {
		if got != expected[name] {
			t.Errorf("%s = %q, want %q", name, got, expected[name])
		}
	}
}

func TestGetenvPrefersCanonicalName(t *testing.T) {
	t.Setenv("GOLANG_CC_LOG_LEVEL", "debug")
	t.Setenv("GOLANG_CLAUDE_CODE_LOG_LEVEL", "info")

	if got := Getenv("GOLANG_CC_LOG_LEVEL"); got != "debug" {
		t.Fatalf("Getenv() = %q, want canonical value", got)
	}
}

func TestGetenvFallsBackToLegacyName(t *testing.T) {
	t.Setenv("GOLANG_CC_PROVIDER", "")
	t.Setenv("GOLANG_CLAUDE_CODE_PROVIDER", "legacy-provider")

	if got := Getenv("GOLANG_CC_PROVIDER"); got != "legacy-provider" {
		t.Fatalf("Getenv() = %q, want legacy value", got)
	}
}

func TestGetenvFallsBackToShortLegacyName(t *testing.T) {
	t.Setenv("GOLANG_CC_WEBSEARCH_URL", "")
	t.Setenv("GOLANG_CLAUDE_CODE_WEBSEARCH_URL", "")
	t.Setenv("GO_CLAUDE_CODE_WEBSEARCH_URL", "legacy-short")

	if got := Getenv("GOLANG_CC_WEBSEARCH_URL"); got != "legacy-short" {
		t.Fatalf("Getenv() = %q, want short legacy value", got)
	}
}

func TestGetenvFallsBackToCompactLegacyName(t *testing.T) {
	t.Setenv("GOLANG_CC_STABLE_PREFIX_SKILLS", "")
	t.Setenv("GOLANG_CLAUDE_CODE_STABLE_PREFIX_SKILLS", "")
	t.Setenv("GO_CLAUDE_CODE_STABLE_PREFIX_SKILLS", "")
	t.Setenv("GO_CLAUDE_STABLE_PREFIX_SKILLS", "1")

	if got := Getenv("GOLANG_CC_STABLE_PREFIX_SKILLS"); got != "1" {
		t.Fatalf("Getenv() = %q, want compact legacy value", got)
	}
}

func TestLookupEnvCanonicalEmptySuppressesLegacy(t *testing.T) {
	t.Setenv("GOLANG_CC_PROVIDER", "")
	t.Setenv("GOLANG_CLAUDE_CODE_PROVIDER", "legacy-provider")

	if got, ok := LookupEnv("GOLANG_CC_PROVIDER"); !ok || got != "" {
		t.Fatalf("LookupEnv() = %q, %v; want explicit canonical empty value", got, ok)
	}
}

func TestPromoteEnvironmentOverridesLegacyWithCanonical(t *testing.T) {
	t.Setenv("GOLANG_CC_TEST_PROMOTION", "canonical")
	t.Setenv("GOLANG_CLAUDE_CODE_TEST_PROMOTION", "legacy")

	if err := PromoteEnvironment(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GOLANG_CLAUDE_CODE_TEST_PROMOTION"); got != "canonical" {
		t.Fatalf("promoted value = %q, want canonical", got)
	}
	if got := os.Getenv("GO_CLAUDE_TEST_PROMOTION"); got != "canonical" {
		t.Fatalf("compact legacy promoted value = %q, want canonical", got)
	}
	if got := os.Getenv("GO_CLAUDE_CODE_TEST_PROMOTION"); got != "canonical" {
		t.Fatalf("short legacy promoted value = %q, want canonical", got)
	}
}

func TestPromoteEnvironmentCopiesLegacyToCanonical(t *testing.T) {
	t.Setenv("GOLANG_CLAUDE_CODE_TEST_LEGACY_PROMOTION", "legacy")
	_ = os.Unsetenv("GOLANG_CC_TEST_LEGACY_PROMOTION")
	t.Cleanup(func() { _ = os.Unsetenv("GOLANG_CC_TEST_LEGACY_PROMOTION") })

	if err := PromoteEnvironment(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GOLANG_CC_TEST_LEGACY_PROMOTION"); got != "legacy" {
		t.Fatalf("canonical value = %q, want legacy fallback", got)
	}
}

func TestPromoteEnvironmentCopiesShortLegacyToCanonical(t *testing.T) {
	t.Setenv("GO_CLAUDE_CODE_TEST_SHORT_PROMOTION", "legacy-short")
	_ = os.Unsetenv("GOLANG_CC_TEST_SHORT_PROMOTION")
	_ = os.Unsetenv("GOLANG_CLAUDE_CODE_TEST_SHORT_PROMOTION")
	_ = os.Unsetenv("GOLANG_CC_CODE_TEST_SHORT_PROMOTION")
	t.Cleanup(func() {
		_ = os.Unsetenv("GOLANG_CC_TEST_SHORT_PROMOTION")
		_ = os.Unsetenv("GOLANG_CLAUDE_CODE_TEST_SHORT_PROMOTION")
		_ = os.Unsetenv("GOLANG_CC_CODE_TEST_SHORT_PROMOTION")
	})

	if err := PromoteEnvironment(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GOLANG_CC_TEST_SHORT_PROMOTION"); got != "legacy-short" {
		t.Fatalf("canonical value = %q, want short legacy fallback", got)
	}
	if value, ok := os.LookupEnv("GOLANG_CC_CODE_TEST_SHORT_PROMOTION"); ok {
		t.Fatalf("overlapping prefix created spurious canonical variable %q", value)
	}
}
