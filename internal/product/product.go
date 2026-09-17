package product

import (
	"os"
	"strings"
)

const (
	Name             = "go-e2e"
	BinaryName       = Name
	ProductKey       = Name
	ConfigDirName    = ".golang-cc"
	GuidanceFilename = "go-e2e.md"

	// These persisted identifiers predate the product rename and must not be
	// derived from Name. Existing transcripts and Redis keys use them.
	TranscriptSchemaV1     = "golang-cc.transcript.v1"
	TranscriptSchemaV2     = "golang-cc.transcript.v2"
	TenantQuotaRedisPrefix = "golang-cc:tenant_quota"
	MobileUsageRedisPrefix = "golang-cc:mobile_usage"

	EnvPrefix              = "GO_E2E_"
	LegacyEnvPrefix        = "GOLANG_CC_"
	LegacyClaudeEnvPrefix  = "GOLANG_CLAUDE_CODE_"
	LegacyShortEnvPrefix   = "GO_CLAUDE_CODE_"
	LegacyCompactEnvPrefix = "GO_CLAUDE_"

	LegacyProductName        = "go-claude"
	LegacyConfigDirName      = ".go-claude"
	LegacyTranscriptSchemaV1 = "go-claude.transcript.v1"
	LegacyTranscriptSchemaV2 = "go-claude.transcript.v2"
	PreviousProductName      = "golang-cc"
	PreviousBinaryName       = "golang-cc"
	PreviousGuidanceFilename = "golang-cc.md"
	LegacyGuidanceFilename   = "go-claude.md"

	ProductNameEnv      = EnvPrefix + "PRODUCT_NAME"
	ProductKeyEnv       = EnvPrefix + "PRODUCT_KEY"
	ConfigDirEnv        = EnvPrefix + "CONFIG_DIR"
	ConfigDirNameEnv    = EnvPrefix + "CONFIG_DIR_NAME"
	GuidanceFilenameEnv = EnvPrefix + "GUIDANCE_FILE"
)

// Getenv reads the canonical environment variable first and falls back through
// every supported product prefix. key may use any supported prefix so call
// sites can migrate incrementally without changing precedence.
func Getenv(key string) string {
	for _, candidate := range envKeys(key) {
		if value := os.Getenv(candidate); value != "" {
			return value
		}
	}
	return ""
}

// LookupEnv preserves explicit empty values while preferring the canonical key.
func LookupEnv(key string) (string, bool) {
	for _, candidate := range envKeys(key) {
		if value, ok := os.LookupEnv(candidate); ok {
			return value, true
		}
	}
	return "", false
}

// PromoteEnvironment makes canonical variables visible to legacy os.Getenv
// call sites. It is called before the CLI initializes; direct package users can
// use Getenv while the remaining call sites are migrated.
func PromoteEnvironment() error {
	for _, legacyPrefix := range compatibilityEnvPrefixes() {
		if err := promoteLegacyPrefix(legacyPrefix); err != nil {
			return err
		}
	}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(key, EnvPrefix) {
			continue
		}
		suffix := strings.TrimPrefix(key, EnvPrefix)
		for _, legacyPrefix := range compatibilityEnvPrefixes() {
			if err := os.Setenv(legacyPrefix+suffix, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func promoteLegacyPrefix(legacyPrefix string) error {
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(key, legacyPrefix) {
			continue
		}
		if legacyPrefix == LegacyCompactEnvPrefix &&
			(strings.HasPrefix(key, LegacyShortEnvPrefix) || strings.HasPrefix(key, LegacyClaudeEnvPrefix)) {
			continue
		}
		canonical := EnvPrefix + strings.TrimPrefix(key, legacyPrefix)
		if _, exists := os.LookupEnv(canonical); exists {
			continue
		}
		if err := os.Setenv(canonical, value); err != nil {
			return err
		}
	}
	return nil
}

func envKeys(key string) []string {
	var suffix string
	switch {
	case strings.HasPrefix(key, EnvPrefix):
		suffix = strings.TrimPrefix(key, EnvPrefix)
	case strings.HasPrefix(key, LegacyEnvPrefix):
		suffix = strings.TrimPrefix(key, LegacyEnvPrefix)
	case strings.HasPrefix(key, LegacyClaudeEnvPrefix):
		suffix = strings.TrimPrefix(key, LegacyClaudeEnvPrefix)
	case strings.HasPrefix(key, LegacyShortEnvPrefix):
		suffix = strings.TrimPrefix(key, LegacyShortEnvPrefix)
	case strings.HasPrefix(key, LegacyCompactEnvPrefix):
		suffix = strings.TrimPrefix(key, LegacyCompactEnvPrefix)
	default:
		return []string{key}
	}
	keys := []string{EnvPrefix + suffix}
	for _, legacyPrefix := range compatibilityEnvPrefixes() {
		keys = append(keys, legacyPrefix+suffix)
	}
	return keys
}

func compatibilityEnvPrefixes() []string {
	return []string{LegacyEnvPrefix, LegacyClaudeEnvPrefix, LegacyShortEnvPrefix, LegacyCompactEnvPrefix}
}
