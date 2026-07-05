package elta_home

import "github.com/chobits-os/chobits/core"

// UpdateDictionarySchema 定义 update_dictionary 工具。
// 大模型用它向自己的记忆扇区写入带上下文的词条。
var UpdateDictionarySchema = core.ToolSchema{
	Type: "function",
	Function: core.FunctionDefinition{
		Name:        "update_dictionary",
		Description: "向你的长期记忆（L2 语义词条）写入一条带上下文的记录。每条记录包含 tag（标签）、content（内容）和 context（上下文/场景）。当你发现用户的重要信息（如生日、喜好、系统配置）时，请主动调用此工具记住它。",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tag": map[string]interface{}{
					"type":        "string",
					"description": "记忆标签，用于检索。例如：'[饮食]', '[生日]', '[系统配置]', '[心情]'",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "记忆内容实体。例如：'主人对海鲜过敏'",
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "★ 上下文：记录这个记忆时的场景描述。例如：'2026年7月5日晚上，主人点外卖时提到的'",
				},
			},
			"required": []string{"tag", "content", "context"},
		},
	},
}

// GetTools 返回 Elta 生活区可用的所有工具列表。
func GetTools() []core.ToolSchema {
	return []core.ToolSchema{
		UpdateDictionarySchema,
	}
}
