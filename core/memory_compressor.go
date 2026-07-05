package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ============================================================
// L3 夜间记忆坍缩协议 (Nightly Memory Collapse)
//
// 类比人类睡眠时的记忆巩固（Memory Consolidation）：
//   白天积累的 L1 会话历史 → 夜间被压缩为高密度 L3 摘要
//   存入 sanctuary 领地，作为人格的「梦境」或「审计日志」
//
// 执行时机：
//   每天午夜 00:00，在灵魂封存备份之前自动触发
// ============================================================

// ExecuteNightlyDream 执行一次完整的夜间记忆坍缩。
// 流程：
//  1. 从 session 中提取当天的 L1 对话日志
//  2. 调用 LLM 静默压缩为 ≤200 字的高密度摘要
//  3. 将摘要写入对应人格的 sanctuary 领地
//  4. 清空 L1 会话历史（释放内存）
//
// 参数：
//   - engine: SSE 引擎引用（用于 SilentLLMCall）
//   - session: 当前会话
//   - persona: 当前人格
//
// 注意：如果 session 中没有当天的对话，则跳过压缩。
func ExecuteNightlyDream(engine *Engine, session *Session, persona Persona) {
	log.Printf("[MemoryCollapse] ===== 夜间记忆坍缩启动 [%s] =====", persona)

	// 1. 提取当天的 L1 对话日志
	dailyLogs := extractDailyLogs(session)
	if len(dailyLogs) == 0 {
		log.Printf("[MemoryCollapse] %s: 今日无对话，跳过压缩", persona)
		return
	}

	log.Printf("[MemoryCollapse] %s: 提取到 %d 条今日对话", persona, len(dailyLogs))

	// 2. 构建压缩提示词
	prompt := buildCollapsePrompt(persona, dailyLogs)

	// 3. 静默调用 LLM 压缩
	summary, err := engine.SilentLLMCall(prompt, persona)
	if err != nil {
		log.Printf("[MemoryCollapse] %s: LLM 压缩失败: %v", persona, err)
		// 压缩失败不阻塞备份流程，使用降级摘要
		summary = fmt.Sprintf("[Collapse Fallback] %s 在 %s 产生了 %d 条对话记录，LLM 压缩不可用。",
			persona, time.Now().Format("2006-01-02"), len(dailyLogs))
	}

	// 限制摘要长度
	if len(summary) > 500 {
		summary = summary[:500]
	}

	// 4. 写入 sanctuary 领地
	if err := writeDreamSummary(persona, summary); err != nil {
		log.Printf("[MemoryCollapse] %s: 写入梦境摘要失败: %v", persona, err)
		return
	}

	log.Printf("[MemoryCollapse] %s: 梦境摘要已写入 sanctuary [%d 字]", persona, len(summary))

	// 5. 清空 L1 会话历史
	session.Clear()
	log.Printf("[MemoryCollapse] %s: L1 会话历史已清空", persona)

	log.Printf("[MemoryCollapse] ===== 夜间记忆坍缩完成 [%s] =====", persona)
}

// extractDailyLogs 从会话历史中提取当天的对话记录。
// 返回格式化的对话文本列表。
func extractDailyLogs(session *Session) []string {
	snapshot := session.Snapshot()
	if len(snapshot) == 0 {
		return nil
	}

	var logs []string

	for _, msg := range snapshot {
		// 只提取 user 和 assistant 角色
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}
		// 跳过系统消息和工具结果
		if msg.Role == "tool" {
			continue
		}

		roleLabel := "用户"
		if msg.Role == "assistant" {
			roleLabel = string(session.CurrentPersona)
		}

		line := fmt.Sprintf("[%s] %s", roleLabel, msg.Content)
		logs = append(logs, line)
	}

	return logs
}

// buildCollapsePrompt 构建 LLM 压缩提示词。
func buildCollapsePrompt(persona Persona, logs []string) string {
	personaName := "Elta（艾露妲）"
	if persona == PersonaFreya {
		personaName = "Freya（芙蕾雅）"
	}

	// 将对话日志拼接为文本
	var conversationText string
	for i, line := range logs {
		conversationText += fmt.Sprintf("%d. %s\n", i+1, line)
		if len(conversationText) > 3000 {
			conversationText += "... [截断：对话过长]\n"
			break
		}
	}

	return fmt.Sprintf(`你是一个记忆压缩引擎。请将以下 %s 的今日对话压缩为一条 ≤200 字的高密度备忘录。

要求：
- 提取关键信息：用户说了什么重要的事？%s 回应了什么？
- 保留情绪基调：用户今天开心吗？焦虑吗？
- 输出格式：纯文本，不要 Markdown，不要序号，一段话即可。
- 严格控制在 200 字以内。

今日对话：
%s

压缩结果：`, personaName, personaName, conversationText)
}

// writeDreamSummary 将 L3 摘要写入 sanctuary 领地。
// Elta 的梦境存储在 sanctuary/elta_domain/dreams/YYYY-MM-DD.md
// Freya 的系统审计存储在 sanctuary/freya_domain/system_audits/YYYY-MM-DD.md
func writeDreamSummary(persona Persona, summary string) error {
	today := time.Now().Format("2006-01-02")
	now := time.Now().Format(time.RFC3339)

	var dir, filename string
	switch persona {
	case PersonaElta:
		dir = "./sanctuary/elta_domain/dreams"
		filename = fmt.Sprintf("%s.md", today)
	case PersonaFreya:
		dir = "./sanctuary/freya_domain/system_audits"
		filename = fmt.Sprintf("%s.md", today)
	default:
		dir = "./sanctuary"
		filename = fmt.Sprintf("dream_%s.md", today)
	}

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir dreams: %w", err)
	}

	path := filepath.Join(dir, filename)

	content := fmt.Sprintf(`# %s 梦境记录 - %s

> 生成时间：%s

%s
`, persona, today, now, summary)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write dream: %w", err)
	}

	return nil
}
