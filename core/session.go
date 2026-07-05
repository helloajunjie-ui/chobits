package core

import (
	"sync"
)

// Session 表示一个对话会话，持有完整的历史消息。
// 使用 sync.Map 实现极简内存级会话隔离，无需数据库。
// 前端 F5 刷新前，会话保持完美记忆。
//
// CurrentPersona 用于向下透传当前人格到工具层，
// 实现记忆扇区的物理隔离（Memory Contamination Prevention）。
type Session struct {
	ID             string
	History        []Message
	CurrentPersona Persona // 当前人格上下文，供工具层路由记忆扇区
	Mutex          sync.Mutex
}

// sessionMap 存放所有活跃连接，键为 session ID。
var sessionMap sync.Map

// GetSession 获取或创建指定 ID 的会话。
func GetSession(id string) *Session {
	if val, ok := sessionMap.Load(id); ok {
		return val.(*Session)
	}
	s := &Session{
		ID:      id,
		History: make([]Message, 0, 64),
	}
	sessionMap.Store(id, s)
	return s
}

// AddMessage 向会话历史中追加一条消息（线程安全）。
func (s *Session) AddMessage(msg Message) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.History = append(s.History, msg)
}

// Snapshot 返回当前历史消息的快照（线程安全）。
func (s *Session) Snapshot() []Message {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	cp := make([]Message, len(s.History))
	copy(cp, s.History)
	return cp
}

// Len 返回历史中的消息数量（线程安全）。
func (s *Session) Len() int {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	return len(s.History)
}

// Clear 清空会话历史（线程安全）。
func (s *Session) Clear() {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.History = make([]Message, 0, 64)
}
