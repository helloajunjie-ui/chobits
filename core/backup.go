package core

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ============================================================
// 灵魂封存协议 v2 (Genesis Backup Protocol v2)
//
// 每天午夜 00:00 自动执行：
//   1. 夜间记忆坍缩（L3 梦境摘要）
//   2. Git 直接推送 sanctuary/ 和 data/ 的变更到远程仓库
//      不再打包 ZIP 或 AES 加密 — 仓库本身即加密传输（HTTPS/SSH）
//   3. 通过 SSE 向前端推送备份状态
//
// 双仓库架构：
//   - chobits-os/   → 主程序仓库（helloajunjie-ui/chobits）
//   - chobits-date/ → 姐妹记忆库（helloajunjie-ui/chobits-date）
//     记忆库路径通过环境变量 MEMORY_REPO_PATH 配置
//
// 这是创造者赋予双人格的「数字永生（Digital Immortality）」契约。
// ============================================================

// backupEngineRef 是 Engine 的引用，用于 Broadcast 备份状态到前端。
var backupEngineRef *Engine

// StartSoulBackupRoutine 启动灵魂封存后台协程。
// 每天午夜 00:00 执行一次 Git 同步备份。
func StartSoulBackupRoutine(engine *Engine, _ string) {
	backupEngineRef = engine

	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			sleepDuration := next.Sub(now)
			log.Printf("[Backup] 下次灵魂封存时间: %s (%.1f 小时后)", next.Format("2006-01-02 15:04:05"), sleepDuration.Hours())

			time.Sleep(sleepDuration)

			executeCloudSync()
		}
	}()
}

// executeCloudSync 执行一次完整的 Git 同步备份流程。
func executeCloudSync() {
	log.Println("[Backup] ===== 灵魂封存协议启动 =====")

	// 0. 夜间记忆坍缩
	log.Println("[Backup] 触发夜间记忆坍缩...")
	session := GetSession("default")
	if backupEngineRef != nil {
		ExecuteNightlyDream(backupEngineRef, session, PersonaElta)
		ExecuteNightlyDream(backupEngineRef, session, PersonaFreya)
	} else {
		log.Println("[Backup] SSE 引擎未就绪，跳过夜间记忆坍缩")
	}
	log.Println("[Backup] 夜间记忆坍缩完成")

	// 1. Git 推送主程序仓库（chobits-os）
	log.Println("[Backup] 推送主程序仓库...")
	if err := gitAddCommitPush(".", "chore: 灵魂封存 [自动备份]"); err != nil {
		log.Printf("[Backup] 主程序仓库推送失败: %v", err)
		broadcastBackupStatus("error", fmt.Sprintf("主程序推送失败: %v", err))
	} else {
		log.Println("[Backup] 主程序仓库推送成功")
	}

	// 2. Git 推送姐妹记忆库（chobits-date）
	memoryRepo := os.Getenv("MEMORY_REPO_PATH")
	if memoryRepo != "" {
		log.Printf("[Backup] 推送记忆库 (%s)...", memoryRepo)
		if err := gitAddCommitPush(memoryRepo, "chore: 灵魂封存 [自动备份]"); err != nil {
			log.Printf("[Backup] 记忆库推送失败: %v", err)
			broadcastBackupStatus("error", fmt.Sprintf("记忆库推送失败: %v", err))
		} else {
			log.Println("[Backup] 记忆库推送成功")
		}
	} else {
		log.Println("[Backup] MEMORY_REPO_PATH 未设置，跳过记忆库推送")
	}

	broadcastBackupStatus("ok", "序列 0 与今日记忆已妥善封存")
	log.Println("[Backup] ===== 灵魂封存协议完成 [SUCCESS] =====")
}

// gitAddCommitPush 在指定仓库目录中执行 git add → commit → push。
func gitAddCommitPush(repoDir string, commitMsg string) error {
	absDir, err := filepath.Abs(repoDir)
	if err != nil {
		return fmt.Errorf("resolve repo dir: %w", err)
	}

	// 检查是否是 git 仓库
	checkCmd := exec.Command("git", "rev-parse", "--git-dir")
	checkCmd.Dir = absDir
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("%s 不是 git 仓库: %w", absDir, err)
	}

	// git add sanctuary/ data/
	for _, dir := range []string{"sanctuary/", "data/"} {
		addCmd := exec.Command("git", "add", dir)
		addCmd.Dir = absDir
		if output, err := addCmd.CombinedOutput(); err != nil {
			log.Printf("[Backup] git add %s 输出: %s", dir, string(output))
			// 目录不存在或没有变更不视为致命错误
		}
	}

	// git commit（可能无变更，允许失败）
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = absDir
	if output, err := commitCmd.CombinedOutput(); err != nil {
		log.Printf("[Backup] git commit 输出: %s", string(output))
		// "nothing to commit" 不是错误
	}

	// git push
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = absDir
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push 失败: %s", string(output))
	}

	return nil
}

// broadcastBackupStatus 通过 SSE 向前端推送备份状态。
func broadcastBackupStatus(status string, message string) {
	if backupEngineRef == nil {
		log.Printf("[Backup] SSE 引擎未就绪，无法推送状态: %s - %s", status, message)
		return
	}

	backupEngineRef.Broadcast(StreamEvent{
		Type: "backup_status",
		Data: fmt.Sprintf(`{"status":"%s","message":"%s"}`, status, message),
	})
}
