package skills

import (
	"github.com/chobits-os/chobits/core"
	"github.com/chobits-os/chobits/skills/elta_home"
	"github.com/chobits-os/chobits/skills/freya_arsenal"
)

// domainTools 返回所有人格共享的主权级文件工具。
// 定义在此处而非 elta_home/freya_arsenal，避免循环依赖。
func domainTools() []core.ToolSchema {
	return []core.ToolSchema{
		DomainFileReadSchema,
		DomainFileWriteSchema,
		DomainFileDeleteSchema,
		DomainDirListSchema,
	}
}

// GetEltaTools 返回 Elta（表人格）可用的所有工具。
func GetEltaTools() []core.ToolSchema {
	tools := make([]core.ToolSchema, 0)

	// 主权级文件工具（沙盒化领地操作）
	tools = append(tools, domainTools()...)

	// Elta 生活区工具
	tools = append(tools, elta_home.GetTools()...)

	// 必须带上求救呼机
	tools = append(tools, elta_home.CallFreyaOverrideSchema)

	return tools
}

// GetFreyaTools 返回 Freya（里人格）军械库的所有工具。
func GetFreyaTools() []core.ToolSchema {
	tools := make([]core.ToolSchema, 0)

	// 主权级文件工具（沙盒化领地操作）
	tools = append(tools, domainTools()...)

	// Freya 军械库工具
	tools = append(tools, freya_arsenal.GetTools()...)

	return tools
}

// Init 初始化技能系统，注册函数指针到 core 包。
// 使用函数指针注册机制避免 core 与 skills 的循环依赖。
func Init() {
	core.GetFreyaTools = GetFreyaTools
	core.DomainFileRead = DomainFileRead
	core.DomainFileWrite = DomainFileWrite
	core.DomainFileDelete = DomainFileDelete
	core.DomainDirList = DomainDirList
}
