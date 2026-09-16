// config.go 负责候选条数与模型档位的解析。模型解析刻意不硬编码 Anthropic
// 模型 id：非 Anthropic 部署拿到不存在的 id 会直接从 provider 报错。

package nextsteps

import (
	"strings"

	"github.com/konglong87/go-e2e/internal/config"
)

// modelTierHaiku 是本功能请求的模型档位。语义与 agentruntime 的子代理档位
// 解析一致：Anthropic 会话映射到具体 haiku 模型，非 Anthropic 会话查
// settings.subagentModelTiers，查不到就继承主模型。
const modelTierHaiku = "haiku"

type Config struct {
	Enabled bool
	Model   string
	Count   int
}

func (c Config) WithDefaults(parentModel string, tierModels map[string]string) Config {
	switch {
	case c.Count <= 0:
		c.Count = DefaultCount
	case c.Count > MaxCount:
		c.Count = MaxCount
	}
	c.Model = resolveModel(c.Model, parentModel, tierModels)
	return c
}

func resolveModel(configured, parentModel string, tierModels map[string]string) string {
	if configured = strings.TrimSpace(configured); configured != "" {
		return configured
	}
	parent := strings.TrimSpace(parentModel)
	// 主模型本来就在这一档，不必再切。
	if strings.Contains(strings.ToLower(parent), modelTierHaiku) {
		return parent
	}
	// 空主模型意味着在用 Anthropic 内建默认值。
	if parent == "" || isAnthropicModel(parent) {
		return defaultHaikuModel()
	}
	if tier := strings.TrimSpace(tierModels[modelTierHaiku]); tier != "" {
		return tier
	}
	return parent
}

// isAnthropicModel 靠 "claude" 子串判定 Anthropic 家族，与 agentruntime 的
// 同名判定一致——这样不必把 provider 配置一路传进来。
func isAnthropicModel(model string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(model)), "claude")
}

func defaultHaikuModel() string {
	for _, model := range config.KnownModels {
		if strings.Contains(strings.ToLower(model), modelTierHaiku) {
			return model
		}
	}
	return config.DefaultModel()
}
