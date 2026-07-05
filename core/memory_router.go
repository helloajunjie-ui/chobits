package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// memorySector 表示一个物理记忆扇区。
// 每个扇区对应一个 JSON 文件，由对应人格独占读写。
// 存储结构化 MemoryEntry 列表，每条记录带 ID/Tag/Content/Context/Timestamp。
type memorySector struct {
	mu      sync.Mutex
	path    string
	persona Persona
	entries []MemoryEntry
}

// dataMount 返回 DATA_MOUNT_POINT 环境变量值，默认 "./data"。
func dataMount() string {
	mount := os.Getenv("DATA_MOUNT_POINT")
	if mount == "" {
		return "./data"
	}
	return mount + "/data"
}

// eltaSector 是 Elta 的白区记忆扇区（心智档案 Heart Archive）。
// 只存情感流、饮食偏好、日程、人际关系。
var eltaSector = &memorySector{
	path:    filepath.Join(dataMount(), "memory_elta/dict.json"),
	persona: PersonaElta,
}

// freyaSector 是 Freya 的黑区记忆扇区（核心寄存器 Core Registers）。
// 只存系统路径、服务器 IP、代码偏好、API 密钥。
var freyaSector = &memorySector{
	path:    filepath.Join(dataMount(), "memory_freya/dict.json"),
	persona: PersonaFreya,
}

// init 在包初始化时加载两个记忆扇区。
func init() {
	eltaSector.load()
	freyaSector.load()
}

// load 从 JSON 文件加载记忆数据到内存。
func (ms *memorySector) load() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	raw, err := os.ReadFile(ms.path)
	if err != nil {
		log.Printf("[MemorySector] 无法加载 %s: %v，使用空数据", ms.path, err)
		ms.entries = make([]MemoryEntry, 0)
		return
	}

	var wrapper struct {
		Persona Persona       `json:"persona"`
		Entries []MemoryEntry `json:"entries"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		log.Printf("[MemorySector] 解析 %s 失败: %v，使用空数据", ms.path, err)
		ms.entries = make([]MemoryEntry, 0)
		return
	}

	if wrapper.Entries == nil {
		ms.entries = make([]MemoryEntry, 0)
	} else {
		ms.entries = wrapper.Entries
	}
	log.Printf("[MemorySector] 已加载 %s (%d 条记录)", ms.path, len(ms.entries))
}

// persist 将内存数据写回 JSON 文件。
func (ms *memorySector) persist() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	wrapper := struct {
		Persona Persona       `json:"persona"`
		Entries []MemoryEntry `json:"entries"`
	}{
		Persona: ms.persona,
		Entries: ms.entries,
	}

	raw, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		log.Printf("[MemorySector] 序列化 %s 失败: %v", ms.path, err)
		return
	}

	if err := os.WriteFile(ms.path, raw, 0644); err != nil {
		log.Printf("[MemorySector] 写入 %s 失败: %v", ms.path, err)
	}
}

// AddEntry 追加一条结构化记忆条目并持久化。
func (ms *memorySector) AddEntry(tag, content, context string) MemoryEntry {
	entry := NewMemoryEntry(tag, content, context)

	ms.mu.Lock()
	ms.entries = append(ms.entries, entry)
	ms.mu.Unlock()

	ms.persist()
	return entry
}

// Get 根据 Tag 查询第一条匹配的记忆记录。
func (ms *memorySector) Get(tag string) (MemoryEntry, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, e := range ms.entries {
		if e.Tag == tag {
			return e, true
		}
	}
	return MemoryEntry{}, false
}

// GetAll 返回当前扇区的全部记忆快照。
func (ms *memorySector) GetAll() []MemoryEntry {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	cp := make([]MemoryEntry, len(ms.entries))
	copy(cp, ms.entries)
	return cp
}

// resolveSector 根据 Persona 返回对应的记忆扇区。
// 这是记忆隔离的核心路由：同一条 update_dictionary 工具调用，
// 根据当前人格被静默分流到不同的物理文件。
func resolveSector(p Persona) *memorySector {
	switch p {
	case PersonaElta:
		return eltaSector
	case PersonaFreya:
		return freyaSector
	default:
		// 防御性编程：未知人格默认走白区
		log.Printf("[MemoryRouter] 警告：未知人格 %s，默认路由到 Elta 扇区", p)
		return eltaSector
	}
}

// ExecuteMemoryUpdate 是工具层调用的记忆写入入口。
// 它根据 session.CurrentPersona 自动路由到对应的物理扇区，
// 大模型不需要知道自己存到了哪个文件里。
//
// 参数：
//   - session: 当前会话（内含 CurrentPersona 用于路由）
//   - tag:     记忆标签（键名）
//   - content: 记忆内容
//   - context: 记忆上下文（情绪、场景描述）
//
// 返回格式化后的结果字符串。
func ExecuteMemoryUpdate(session *Session, tag string, content string, context string) string {
	sector := resolveSector(session.CurrentPersona)

	var prefixLog string
	switch session.CurrentPersona {
	case PersonaElta:
		prefixLog = "[Elta Heart]"
	case PersonaFreya:
		prefixLog = "[Freya Register]"
	default:
		prefixLog = "[MemoryRouter]"
	}

	entry := sector.AddEntry(tag, content, context)
	log.Printf("%s Writing to sector: %s -> %s (ctx: %s) [id=%s]\n", prefixLog, tag, content, context, entry.ID)

	return fmt.Sprintf(`{"status":"ok","sector":"%s","tag":"%s","id":"%s"}`,
		session.CurrentPersona, tag, entry.ID)
}

// ExecuteMemoryQuery 查询记忆扇区中的一条记录。
func ExecuteMemoryQuery(session *Session, tag string) string {
	sector := resolveSector(session.CurrentPersona)

	entry, ok := sector.Get(tag)
	if !ok {
		return fmt.Sprintf(`{"status":"not_found","tag":"%s"}`, tag)
	}
	return fmt.Sprintf(`{"status":"ok","tag":"%s","content":"%s","context":"%s","timestamp":"%s"}`,
		entry.Tag, entry.Content, entry.Context, entry.Timestamp)
}

// ExecuteMemoryDump 返回当前人格的完整记忆快照（用于前端展示）。
func ExecuteMemoryDump(session *Session) []MemoryEntry {
	sector := resolveSector(session.CurrentPersona)
	return sector.GetAll()
}
