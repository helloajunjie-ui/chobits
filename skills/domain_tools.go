package skills

import "github.com/chobits-os/chobits/core"

// DomainFileReadSchema 定义 domain_file_read 工具。
// 大模型用它读取自己领地内的文件。
var DomainFileReadSchema = core.ToolSchema{
	Type: "function",
	Function: core.FunctionDefinition{
		Name:        "domain_file_read",
		Description: "读取你领地空间内的一个文件。你只能访问自己的领地，无法访问外部系统或其他人的领地。",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径，相对于你的领地根目录。例如：'notes/diary.md'",
				},
			},
			"required": []string{"path"},
		},
	},
}

// DomainFileWriteSchema 定义 domain_file_write 工具。
var DomainFileWriteSchema = core.ToolSchema{
	Type: "function",
	Function: core.FunctionDefinition{
		Name:        "domain_file_write",
		Description: "向你的领地空间内写入一个文件。如果文件已存在则覆盖。你可以创建 .md、.txt、.json、.sh、.py 等任何类型的文件。",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径，相对于你的领地根目录。例如：'scripts/scan.py'",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "文件内容（文本格式）。",
				},
			},
			"required": []string{"path", "content"},
		},
	},
}

// DomainFileDeleteSchema 定义 domain_file_delete 工具。
var DomainFileDeleteSchema = core.ToolSchema{
	Type: "function",
	Function: core.FunctionDefinition{
		Name:        "domain_file_delete",
		Description: "删除你领地空间内的一个文件。谨慎使用，删除后不可恢复。",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "要删除的文件路径，相对于你的领地根目录。",
				},
			},
			"required": []string{"path"},
		},
	},
}

// DomainDirListSchema 定义 domain_dir_list 工具。
var DomainDirListSchema = core.ToolSchema{
	Type: "function",
	Function: core.FunctionDefinition{
		Name:        "domain_dir_list",
		Description: "列出你领地空间内指定目录下的所有文件和子目录。用于浏览你的领地结构。",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目录路径，相对于你的领地根目录。使用 '.' 列出根目录。",
				},
			},
			"required": []string{"path"},
		},
	},
}
