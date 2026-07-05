package elta_home

import "github.com/chobits-os/chobits/core"

// CallFreyaOverrideSchema 是连接表里人格的物理桥梁。
// 艾露妲（Elta）在遇到无法处理的极客任务时，通过此工具呼叫芙蕾雅（Freya）接管。
var CallFreyaOverrideSchema = core.ToolSchema{
	Type: "function",
	Function: core.FunctionDefinition{
		Name:        "call_freya_override",
		Description: "当用户要求执行系统级操作、运行代码、下载文件、爬取网络数据，或你现有工具无法解决的极客任务时，立刻调用此工具呼叫姐姐芙蕾雅接管。绝对不要试图自己解答你能力范围外的问题。",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "呼叫芙蕾雅的原因，例如：'需要使用 yt-dlp 下载视频'",
				},
			},
			"required": []string{"reason"},
		},
	},
}
