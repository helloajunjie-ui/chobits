package core

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

// ToolCall 表示 LLM 返回的工具调用请求。
type ToolCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串
}

// ToolSchema 描述一个工具的 JSON Schema（OpenAI 工具格式）。
type ToolSchema struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 描述一个函数的元信息。
type FunctionDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// Persona 双生子人格枚举。
type Persona string

const (
	PersonaElta  Persona = "ELTA"  // 表人格：生活管家 (Light Mode)
	PersonaFreya Persona = "FREYA" // 里人格：系统极客 (Dark Mode)
)

// StreamEvent SSE 推送的事件结构。
type StreamEvent struct {
	Type          string  `json:"type"` // text, tool_start, tool_end, persona_switch
	Data          string  `json:"data"`
	ActivePersona Persona `json:"active_persona"` // 当前是 ELTA 还是 FREYA 在说话
}

// PersonaProvider 所有人格面具必须实现的接口。
type PersonaProvider interface {
	Name() Persona
	SystemPrompt() string
}

// Router 管理 Elta 与 Freya 的人格跃迁协议。
type Router struct {
	mu       sync.RWMutex
	engine   *Engine
	personas map[Persona]PersonaProvider
	active   Persona
}

// NewRouter 创建并返回一个新的 Router 实例。
func NewRouter(engine *Engine) *Router {
	return &Router{
		engine:   engine,
		personas: make(map[Persona]PersonaProvider),
		active:   PersonaElta, // 默认 Elta
	}
}

// Register 注册一个人格到路由器中。
func (r *Router) Register(p PersonaProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.personas[p.Name()] = p
	log.Printf("[Router] persona registered: %s", p.Name())
}

// Activate 激活指定人格，并广播人格切换事件。
func (r *Router) Activate(p Persona) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.personas[p]; !ok {
		log.Printf("[Router] persona %s not registered", p)
		return false
	}

	if r.active == p {
		return true // 已经是该人格，无需切换
	}

	old := r.active
	r.active = p

	log.Printf("[Router] persona switch: %s -> %s", old, p)

	// 广播人格切换事件
	r.engine.Broadcast(StreamEvent{
		Type:          "persona_switch",
		Data:          string(p),
		ActivePersona: p,
	})

	return true
}

// Active 返回当前激活的人格。
func (r *Router) Active() Persona {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active
}

// ActivePrompt 返回当前激活人格的 System Prompt。
func (r *Router) ActivePrompt() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.personas[r.active]; ok {
		return p.SystemPrompt()
	}
	return ""
}

// SwitchHandler 处理 HTTP 人格切换请求。
// POST /api/persona/switch  body: {"persona":"FREYA"}
func (r *Router) SwitchHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Persona Persona `json:"persona"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !r.Activate(body.Persona) {
		http.Error(w, "persona not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_persona": r.Active(),
	})
}

// HandleToolCall 拦截 LLM 的工具调用请求。
// 如果检测到 call_freya_override，执行人格跃迁协议。
// 返回 true 表示已拦截处理（调用方应阻断当前 Elta 循环），false 表示正常放行。
//
// 注意：此方法保留作为 HTTP API 层的快捷拦截。
// 在 StreamReActLoop 内部已实现完整的拦截逻辑，此处保持向后兼容。
func (r *Router) HandleToolCall(tc ToolCall, originalInput string) bool {
	if tc.Name != "call_freya_override" {
		return false // 不是跃迁调用，放行
	}

	log.Printf("[Router] Freya override requested. reason: %s", tc.Arguments)

	// 1. 广播人格切换事件 — 触发前端日蚀动画
	r.engine.Broadcast(StreamEvent{
		Type:          "persona_switch",
		ActivePersona: PersonaFreya,
		Data:          "init_eclipse",
	})

	// 2. 核心状态接管：切换到 Freya
	r.Activate(PersonaFreya)

	// 3. Freya 硬核启机日志（脑髓切换日志）
	r.engine.Broadcast(StreamEvent{
		Type:          "text",
		ActivePersona: PersonaFreya,
		Data:          "\n\n`[System] Incoming override request detected.`\n",
	})
	r.engine.Broadcast(StreamEvent{
		Type:          "text",
		ActivePersona: PersonaFreya,
		Data:          "`[System] Terminating DeepSeek connection (Elta)... [OK]`\n",
	})
	r.engine.Broadcast(StreamEvent{
		Type:          "text",
		ActivePersona: PersonaFreya,
		Data:          "`[System] Booting Gemini-1.5-Pro core engine... [OK]`\n",
	})
	r.engine.Broadcast(StreamEvent{
		Type:          "text",
		ActivePersona: PersonaFreya,
		Data:          "`[System] Arsenal tools mounted.`\n\n",
	})
	r.engine.Broadcast(StreamEvent{
		Type:          "text",
		ActivePersona: PersonaFreya,
		Data:          "`[Freya] 引擎已切换。说吧，要撕开哪道防火墙？`\n\n",
	})

	// 4. 获取 Freya 军械库工具，启动 ReAct 循环（自动使用 Gemini 脑髓）
	freyaTools := GetFreyaTools()
	if freyaTools == nil {
		r.engine.Broadcast(StreamEvent{
			Type:          "text",
			ActivePersona: PersonaFreya,
			Data:          "`[Freya] 军械库未初始化，请确保 skills.Init() 已被调用。`",
		})
		return true
	}

	session := GetSession("override-" + originalInput)
	go r.engine.StreamReActLoop(originalInput, PersonaFreya, session, r, freyaTools)

	return true // 阻断 Elta 的当前循环
}

// CurrentHandler 返回当前激活的人格。
// GET /api/persona/current
func (r *Router) CurrentHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_persona": r.Active(),
	})
}
