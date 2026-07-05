package freya_arsenal

import "github.com/chobits-os/chobits/core"

// GetTools 返回 Freya 军械库可用的所有工具列表。
func GetTools() []core.ToolSchema {
	return []core.ToolSchema{
		// TODO: 后续添加 Shell 执行、网络爬虫、文件操作等重型工具
	}
}
