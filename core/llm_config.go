package core

import "os"

// BrainConfig 脑髓配置契约。
// 存储与大模型通信所需的连接信息。
type BrainConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

// Valid 检查配置是否有效（API Key 不为空）。
func (c *BrainConfig) Valid() bool {
	return c.APIKey != "" && c.BaseURL != "" && c.Model != ""
}

// GetBrainConfig 异构大脑路由器。
// 根据当前人格枚举，实时组装对应的 LLM 客户端配置：
//   - Elta (表人格) → DeepSeek（中文 NLP 丝滑，共情能力强）
//   - Freya (里人格) → Gemini（超长上下文，狂暴工具调用执行力）
func GetBrainConfig(p Persona) BrainConfig {
	if p == PersonaElta {
		return BrainConfig{
			BaseURL: getEnvOrDefault("ELTA_API_BASE", "https://api.deepseek.com/v1"),
			APIKey:  os.Getenv("ELTA_API_KEY"),
			Model:   getEnvOrDefault("ELTA_MODEL", "deepseek-chat"),
		}
	}

	// Freya → Gemini
	return BrainConfig{
		BaseURL: getEnvOrDefault("FREYA_API_BASE", "https://generativelanguage.googleapis.com/v1beta/openai/"),
		APIKey:  os.Getenv("FREYA_API_KEY"),
		Model:   getEnvOrDefault("FREYA_MODEL", "gemini-1.5-pro"),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
