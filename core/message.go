package core

// Message 表示对话中的一条消息。
type Message struct {
	Role    string `json:"role"`    // system, user, assistant, tool
	Content string `json:"content"` // 文本内容
	Name    string `json:"name,omitempty"`
}

// History 是对话历史的封装。
type History struct {
	Messages []Message
}

// NewHistory 创建并返回一个空的对话历史。
func NewHistory() *History {
	return &History{
		Messages: make([]Message, 0, 64),
	}
}

// Add 向历史中添加一条消息。
func (h *History) Add(msg Message) {
	h.Messages = append(h.Messages, msg)
}

// AddUser 添加一条用户消息。
func (h *History) AddUser(content string) {
	h.Add(Message{Role: "user", Content: content})
}

// AddAssistant 添加一条助手消息。
func (h *History) AddAssistant(content string) {
	h.Add(Message{Role: "assistant", Content: content})
}

// AddTool 添加一条工具结果消息。
func (h *History) AddTool(name, content string) {
	h.Add(Message{Role: "tool", Content: content, Name: name})
}

// AddSystem 添加一条系统消息。
func (h *History) AddSystem(content string) {
	h.Add(Message{Role: "system", Content: content})
}

// Snapshot 返回当前消息列表的快照。
func (h *History) Snapshot() []Message {
	cp := make([]Message, len(h.Messages))
	copy(cp, h.Messages)
	return cp
}

// Len 返回历史中的消息数量。
func (h *History) Len() int {
	return len(h.Messages)
}
