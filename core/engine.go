package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

// Engine 是 Chobits OS 的 SSE 状态流推送引擎。
// 它维护一组订阅的客户端，并在事件发生时广播给所有客户端。
// ctx 用于控制 LLM 流式请求的生命周期（如优雅关闭时取消）。
type Engine struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewEngine 创建并返回一个新的 Engine 实例。
func NewEngine() *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		clients: make(map[chan []byte]struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Subscribe 注册一个新的 SSE 客户端通道，返回该通道。
func (e *Engine) Subscribe() chan []byte {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := make(chan []byte, 64)
	e.clients[ch] = struct{}{}
	log.Printf("[Engine] client subscribed (total: %d)", len(e.clients))
	return ch
}

// Unsubscribe 移除一个 SSE 客户端通道。
func (e *Engine) Unsubscribe(ch chan []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.clients, ch)
	close(ch)
	log.Printf("[Engine] client unsubscribed (total: %d)", len(e.clients))
}

// Broadcast 向所有订阅的客户端广播一个事件。
func (e *Engine) Broadcast(event StreamEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[Engine] marshal error: %v", err)
		return
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for ch := range e.clients {
		select {
		case ch <- data:
		default:
			// 客户端消费太慢，跳过
		}
	}
}

// SSEHandler 处理来自前端的 SSE 连接请求。
func (e *Engine) SSEHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := e.Subscribe()
	defer e.Unsubscribe(ch)

	// 发送初始连接确认
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"data\":\"Chobits OS SSE established\"}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

// SilentLLMCall 静默调用 LLM，不向前端广播任何事件。
// 用于后台任务（如夜间记忆坍缩）中调用 LLM 进行文本处理。
// 参数：
//   - prompt: 发送给 LLM 的提示词
//   - persona: 使用哪个人格的脑配置（Elta=DeepSeek, Freya=Gemini）
//
// 返回 LLM 的完整响应文本。
func (e *Engine) SilentLLMCall(prompt string, persona Persona) (string, error) {
	brain := GetBrainConfig(persona)
	if !brain.Valid() {
		return "", fmt.Errorf("brain config for %s is invalid", persona)
	}

	client := newOpenAIClient(&brain)

	resp, err := client.CreateChatCompletion(
		e.ctx,
		openai.ChatCompletionRequest{
			Model: brain.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: 512,
		},
	)
	if err != nil {
		return "", fmt.Errorf("silent llm call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("silent llm call: no choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}
