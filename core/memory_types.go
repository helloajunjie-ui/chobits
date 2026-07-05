package core

import (
	"time"
)

// ============================================================
// 三维记忆矩阵 (3D Memory Matrix)
//
// L1 — 活跃记忆 (Working Memory): Session.History (现有)
// L2 — 语义词条 (Semantic Dictionary): MemoryEntry 数组
// L3 — 情景压缩 (Episodic Summaries): 夜间记忆坍缩产物
// ============================================================

// MemoryEntry 表示一条带上下文的记忆词条。
// 这是 L2 语义词条的基本单元，替代了旧的 map[string]string。
type MemoryEntry struct {
	ID        string `json:"id"`        // 唯一主键（UUID 或时间戳哈希）
	Tag       string `json:"tag"`       // 标签：[饮食]、[工作]、[核心机密]、[心情]
	Content   string `json:"content"`   // 记忆实体："主人对海鲜过敏"
	Context   string `json:"context"`   // ★ 上下文："2026年7月5日晚上，主人点外卖时提到的"
	Timestamp string `json:"timestamp"` // 写入时间 ISO 8601
}

// NewMemoryEntry 创建一条新的记忆词条。
// 自动生成时间戳和 ID。
func NewMemoryEntry(tag, content, context string) MemoryEntry {
	now := time.Now()
	return MemoryEntry{
		ID:        now.Format("150405.000") + "-" + tag,
		Tag:       tag,
		Content:   content,
		Context:   context,
		Timestamp: now.Format(time.RFC3339),
	}
}

// MemorySector 表示一个物理记忆扇区。
// 每个扇区对应一个 JSON 文件，由对应人格独占读写。
type MemorySector struct {
	Persona Persona       `json:"persona"` // 所属人格
	Entries []MemoryEntry `json:"entries"` // 词条数组（有序）
}

// L3Summary 表示一条 L3 情景压缩（夜间梦境总结）。
type L3Summary struct {
	Date      string  `json:"date"`       // 日期：2026-07-05
	Persona   Persona `json:"persona"`    // 所属人格
	Summary   string  `json:"summary"`    // 高密度备忘录（≤200 字）
	CreatedAt string  `json:"created_at"` // 创建时间
}
