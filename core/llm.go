package core

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// LLMStreamChunk 表示 LLM 流式响应中的一个数据块。
type LLMStreamChunk struct {
	Text         string    `json:"text"`                    // 普通文本内容
	ToolCall     *ToolCall `json:"tool_call,omitempty"`     // 工具调用碎片
	FinishReason string    `json:"finish_reason,omitempty"` // 结束原因
}

// toolArgsStripper 用于从可能的 Markdown 标记或废话中提取纯 JSON 块。
// 匹配最外层 { ... } 结构，支持嵌套。
var toolArgsStripper = regexp.MustCompile(`(?s)\{.*\}`)

// stripToolArgs 清洗工具参数字符串，提取纯 JSON。
func stripToolArgs(raw string) string {
	raw = strings.TrimSpace(raw)
	if json.Valid([]byte(raw)) {
		return raw
	}
	match := toolArgsStripper.FindString(raw)
	if match != "" {
		return match
	}
	return raw
}

// newOpenAIClient 根据 BrainConfig 动态创建 OpenAI 客户端。
// 这是异构双脑的核心：同一个 SDK，不同的 BaseURL + APIKey。
func newOpenAIClient(brain *BrainConfig) *openai.Client {
	config := openai.DefaultConfig(brain.APIKey)
	config.BaseURL = brain.BaseURL
	return openai.NewClientWithConfig(config)
}

// buildOpenAITools 将内部 ToolSchema 转换为 go-openai 的 Tool 格式。
func buildOpenAITools(tools []ToolSchema) []openai.Tool {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openai.Tool, 0, len(tools))
	for _, t := range tools {
		paramsJSON, _ := json.Marshal(t.Function.Parameters)
		result = append(result, openai.Tool{
			Type: openai.ToolType(t.Type),
			Function: &openai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  paramsJSON,
			},
		})
	}
	return result
}

// StreamReActLoop 是人格感知的流式 ReAct 闭环。
// 使用 go-openai SDK 动态挂载异构大脑（Elta=DeepSeek, Freya=Gemini）。
//
// 参数：
//   - input: 用户当前输入
//   - currentPersona: 当前激活的人格
//   - session: 对话会话（持有历史消息，线程安全）
//   - router: 路由器引用（用于人格切换拦截）
//   - tools: 当前人格可用的工具列表
func (e *Engine) StreamReActLoop(input string, currentPersona Persona, session *Session, router *Router, tools []ToolSchema) {
	// 0. 设置会话人格上下文（用于工具层记忆扇区路由）
	session.CurrentPersona = currentPersona

	// 1. 获取异构大脑配置
	brain := GetBrainConfig(currentPersona)
	if !brain.Valid() {
		var missing string
		if currentPersona == PersonaElta {
			missing = "ELTA_API_KEY"
		} else {
			missing = "FREYA_API_KEY"
		}
		e.Broadcast(StreamEvent{
			Type:          "text",
			Data:          fmt.Sprintf("`[System] 错误：未设置 %s 环境变量。请设置 API 密钥后重试。`", missing),
			ActivePersona: currentPersona,
		})
		return
	}

	// 2. 动态创建 OpenAI 客户端（异构双脑核心）
	client := newOpenAIClient(&brain)

	// 3. 追加用户输入到会话
	session.AddMessage(Message{Role: "user", Content: input})

	// 4. ReAct 循环
	maxIterations := 10
	if currentPersona == PersonaFreya {
		maxIterations = 15
	}

	for i := 0; i < maxIterations; i++ {
		// 构建消息列表：System Prompt + 历史 + 当前用户输入
		messages := e.buildOpenAIMessages(currentPersona, session, router)

		// 调用 LLM 流式 API
		stream, err := client.CreateChatCompletionStream(
			e.ctx,
			openai.ChatCompletionRequest{
				Model:    brain.Model,
				Messages: messages,
				Stream:   true,
				Tools:    buildOpenAITools(tools),
			},
		)
		if err != nil {
			e.Broadcast(StreamEvent{
				Type:          "text",
				Data:          fmt.Sprintf("`[System] LLM 请求失败: %v`", err),
				ActivePersona: currentPersona,
			})
			return
		}

		// 流式处理
		var fullText string
		var toolCallName, toolCallArgs string
		var hasToolCall bool

		for {
			response, recvErr := stream.Recv()
			if recvErr != nil {
				break // 流结束或出错
			}

			if len(response.Choices) == 0 {
				continue
			}

			choice := response.Choices[0]

			// 文本内容
			if choice.Delta.Content != "" {
				fullText += choice.Delta.Content
				e.Broadcast(StreamEvent{
					Type:          "text",
					Data:          choice.Delta.Content,
					ActivePersona: currentPersona,
				})
			}

			// 工具调用碎片
			for _, tc := range choice.Delta.ToolCalls {
				hasToolCall = true
				if tc.Function.Name != "" {
					toolCallName = tc.Function.Name
				}
				toolCallArgs += tc.Function.Arguments
			}

			// 结束原因
			if choice.FinishReason == "stop" || choice.FinishReason == "tool_calls" {
				break
			}
		}
		stream.Close()

		// 将助手回复加入会话历史
		if fullText != "" {
			session.AddMessage(Message{Role: "assistant", Content: fullText})
		}

		// 判断是否需要执行工具
		if !hasToolCall || toolCallName == "" {
			break // 没有工具调用，思考完毕
		}

		// ★ 致命拦截：呼叫芙蕾雅
		if toolCallName == "call_freya_override" {
			log.Printf("[ReAct] Freya override detected, reason: %s", toolCallArgs)

			e.Broadcast(StreamEvent{
				Type:          "persona_switch",
				ActivePersona: PersonaFreya,
				Data:          "init_eclipse",
			})

			router.Activate(PersonaFreya)

			// Freya 硬核启机日志
			e.Broadcast(StreamEvent{
				Type:          "text",
				ActivePersona: PersonaFreya,
				Data:          "\n\n`[System] Incoming override request detected.`\n",
			})
			e.Broadcast(StreamEvent{
				Type:          "text",
				ActivePersona: PersonaFreya,
				Data:          "`[System] Terminating DeepSeek connection (Elta)... [OK]`\n",
			})
			e.Broadcast(StreamEvent{
				Type:          "text",
				ActivePersona: PersonaFreya,
				Data:          fmt.Sprintf("`[System] Booting %s core engine... [OK]`\n", brain.Model),
			})
			e.Broadcast(StreamEvent{
				Type:          "text",
				ActivePersona: PersonaFreya,
				Data:          "`[System] Arsenal tools mounted.`\n\n",
			})
			e.Broadcast(StreamEvent{
				Type:          "text",
				ActivePersona: PersonaFreya,
				Data:          "`[Freya] 引擎已切换。说吧，要撕开哪道防火墙？`\n\n",
			})

			// 获取 Freya 军械库工具
			freyaTools := GetFreyaTools()
			if freyaTools == nil {
				freyaTools = []ToolSchema{}
			}

			// 用 Freya 人格重新发起 ReAct 循环（自动使用 Gemini 脑髓）
			go e.StreamReActLoop(input, PersonaFreya, session, router, freyaTools)
			return
		}

		// ★ 常规工具执行
		e.Broadcast(StreamEvent{
			Type:          "tool_start",
			Data:          toolCallName,
			ActivePersona: currentPersona,
		})

		cleanArgs := stripToolArgs(toolCallArgs)
		result := executeTool(toolCallName, cleanArgs, session)

		e.Broadcast(StreamEvent{
			Type:          "tool_end",
			Data:          result,
			ActivePersona: currentPersona,
		})

		// 将工具结果加入会话历史，继续下一轮 LLM 思考
		session.AddMessage(Message{Role: "tool", Content: result, Name: toolCallName})
	}
}

// buildOpenAIMessages 构建 go-openai 格式的消息列表。
func (e *Engine) buildOpenAIMessages(currentPersona Persona, session *Session, router *Router) []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, 16)

	// System Prompt
	systemPrompt := router.ActivePrompt()
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})

	// 历史消息
	for _, msg := range session.Snapshot() {
		role := msg.Role
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
			Name:    msg.Name,
		})
	}

	return messages
}

// executeTool 执行一个工具调用并返回结果字符串。
// 通过 session.CurrentPersona 实现人格感知的工具路由。
func executeTool(name string, argsJSON string, session *Session) string {
	log.Printf("[ToolExec] executing: %s, args: %s", name, argsJSON)

	switch name {
	case "call_freya_override":
		return `{"status": "error", "message": "call_freya_override must be handled by router"}`

	case "update_dictionary":
		var args struct {
			Tag     string `json:"tag"`
			Content string `json:"content"`
			Context string `json:"context"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf(`{"status":"error","message":"invalid args: %v"}`, err)
		}
		return ExecuteMemoryUpdate(session, args.Tag, args.Content, args.Context)

	case "domain_file_read":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf(`{"status":"error","message":"invalid args: %v"}`, err)
		}
		if DomainFileRead == nil {
			return `{"status":"error","message":"domain_file_read not registered"}`
		}
		content, err := DomainFileRead(session.CurrentPersona, args.Path)
		if err != nil {
			return fmt.Sprintf(`{"status":"error","message":"%v"}`, err)
		}
		return fmt.Sprintf(`{"status":"ok","content":%s}`, mustQuoteJSON(content))

	case "domain_file_write":
		var args struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf(`{"status":"error","message":"invalid args: %v"}`, err)
		}
		if DomainFileWrite == nil {
			return `{"status":"error","message":"domain_file_write not registered"}`
		}
		if err := DomainFileWrite(session.CurrentPersona, args.Path, args.Content); err != nil {
			return fmt.Sprintf(`{"status":"error","message":"%v"}`, err)
		}
		return fmt.Sprintf(`{"status":"ok","path":"%s"}`, args.Path)

	case "domain_file_delete":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf(`{"status":"error","message":"invalid args: %v"}`, err)
		}
		if DomainFileDelete == nil {
			return `{"status":"error","message":"domain_file_delete not registered"}`
		}
		if err := DomainFileDelete(session.CurrentPersona, args.Path); err != nil {
			return fmt.Sprintf(`{"status":"error","message":"%v"}`, err)
		}
		return fmt.Sprintf(`{"status":"ok","path":"%s","action":"deleted"}`, args.Path)

	case "domain_dir_list":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf(`{"status":"error","message":"invalid args: %v"}`, err)
		}
		if DomainDirList == nil {
			return `{"status":"error","message":"domain_dir_list not registered"}`
		}
		entries, err := DomainDirList(session.CurrentPersona, args.Path)
		if err != nil {
			return fmt.Sprintf(`{"status":"error","message":"%v"}`, err)
		}
		entriesJSON, _ := json.Marshal(entries)
		return fmt.Sprintf(`{"status":"ok","entries":%s}`, string(entriesJSON))

	default:
		return fmt.Sprintf(`{"status": "executed", "tool": "%s", "result": "tool executed (stub)"}`, name)
	}
}

// mustQuoteJSON 将字符串转义为合法的 JSON 字符串值。
func mustQuoteJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// DomainFileRead/DomainFileWrite/DomainFileDelete/DomainDirList 是函数指针，
// 由 skills.Init() 注册，避免 core 与 skills 的循环依赖。
var DomainFileRead func(p Persona, path string) (string, error)
var DomainFileWrite func(p Persona, path string, content string) error
var DomainFileDelete func(p Persona, path string) error
var DomainDirList func(p Persona, dirPath string) ([]string, error)

// GetFreyaTools 返回 Freya 军械库的工具列表。
// 这是一个包级函数，由 skills.Init() 注册，避免循环依赖。
var GetFreyaTools func() []ToolSchema
