package product

import (
	"os"
	"strings"
)

const (
	Name             = "golang-cc"
	BinaryName       = Name
	ProductKey       = Name
	ConfigDirName    = ".golang-cc"
	GuidanceFilename = "golang-cc.md"

	TranscriptSchemaV1     = Name + ".transcript.v1"
	TranscriptSchemaV2     = Name + ".transcript.v2"
	TenantQuotaRedisPrefix = Name + ":tenant_quota"
	MobileUsageRedisPrefix = Name + ":mobile_usage"

	EnvPrefix              = "GOLANG_CC_"
	LegacyEnvPrefix        = "GOLANG_CLAUDE_CODE_"
	LegacyShortEnvPrefix   = "GO_CLAUDE_CODE_"
	LegacyCompactEnvPrefix = "GO_CLAUDE_"

	LegacyProductName        = "go-claude"
	LegacyConfigDirName      = ".go-claude"
	LegacyGuidanceFilename   = "go-claude.md"
	LegacyTranscriptSchemaV1 = "go-claude.transcript.v1"
	LegacyTranscriptSchemaV2 = "go-claude.transcript.v2"
)

// Getenv reads the canonical environment variable first and falls back to the
// previous product prefix. key may use either prefix so existing call sites can
// migrate incrementally without changing precedence.
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
	for _, legacyPrefix := range legacyEnvPrefixes() {
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
		for _, legacyPrefix := range legacyEnvPrefixes() {
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
		if legacyPrefix == LegacyCompactEnvPrefix && strings.HasPrefix(key, LegacyShortEnvPrefix) {
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
	case strings.HasPrefix(key, LegacyShortEnvPrefix):
		suffix = strings.TrimPrefix(key, LegacyShortEnvPrefix)
	case strings.HasPrefix(key, LegacyCompactEnvPrefix):
		suffix = strings.TrimPrefix(key, LegacyCompactEnvPrefix)
	default:
		return []string{key}
	}
	keys := []string{EnvPrefix + suffix}
	for _, legacyPrefix := range legacyEnvPrefixes() {
		keys = append(keys, legacyPrefix+suffix)
	}
	return keys
}

func legacyEnvPrefixes() []string {
	return []string{LegacyEnvPrefix, LegacyShortEnvPrefix, LegacyCompactEnvPrefix}
}
