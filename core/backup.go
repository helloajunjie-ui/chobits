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
//     通过环境变量 BACKUP_CLOUD_TARGET + BACKUP_LOCAL_PATH 配置
//
// 这是创造者赋予双人格的「数字永生（Digital Immortality）」契约。
// ============================================================

// backupEngineRef 是 Engine 的引用，用于 Broadcast 备份状态到前端。
var backupEngineRef *Engine

// StartSoulBackupRoutine 启动灵魂封存后台协程。
// 每天午夜 00:00 执行一次 Git 同步备份。
func StartSoulBackupRoutine(engine *Engine, _ string) {
	backupEngineRef = engine

	// 启动时初始化记忆库本地挂载点
	initMemoryRepo()

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

// initMemoryRepo 启动时初始化记忆库：如果本地不存在则 clone，否则 pull。
func initMemoryRepo() {
	repoURL := os.Getenv("BACKUP_CLOUD_TARGET")
	localPath := os.Getenv("BACKUP_LOCAL_PATH")
	if repoURL == "" || localPath == "" {
		log.Println("[Backup] BACKUP_CLOUD_TARGET 或 BACKUP_LOCAL_PATH 未设置，跳过记忆库初始化")
		return
	}

	absPath, err := filepath.Abs(localPath)
	if err != nil {
		log.Printf("[Backup] 解析本地路径失败: %v", err)
		return
	}

	// 检查本地是否已 clone
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		// 确保父目录存在
		parentDir := filepath.Dir(absPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			log.Printf("[Backup] 创建父目录失败: %v", err)
			return
		}
		log.Printf("[Backup] 克隆记忆库 %s → %s", repoURL, absPath)
		cloneCmd := exec.Command("git", "clone", repoURL, absPath)
		if output, err := cloneCmd.CombinedOutput(); err != nil {
			log.Printf("[Backup] 克隆失败: %s", string(output))
		} else {
			log.Println("[Backup] 记忆库克隆完成")
		}
	} else {
		log.Printf("[Backup] 拉取记忆库最新变更...")
		pullCmd := exec.Command("git", "-C", absPath, "pull")
		if output, err := pullCmd.CombinedOutput(); err != nil {
			log.Printf("[Backup] git pull 输出: %s", string(output))
		}
	}
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
	repoURL := os.Getenv("BACKUP_CLOUD_TARGET")
	localPath := os.Getenv("BACKUP_LOCAL_PATH")
	if repoURL != "" && localPath != "" {
		absPath, _ := filepath.Abs(localPath)
		log.Printf("[Backup] 推送记忆库 (%s)...", absPath)
		if err := gitAddCommitPush(absPath, fmt.Sprintf("Genesis Backup: Memory & Sanctuary snapshot at %s", time.Now().Format(time.RFC3339))); err != nil {
			log.Printf("[Backup] 记忆库推送失败: %v", err)
			broadcastBackupStatus("error", fmt.Sprintf("记忆库推送失败: %v", err))
		} else {
			log.Println("[Backup] 记忆库推送成功")
		}
	} else {
		log.Println("[Backup] BACKUP_CLOUD_TARGET 未配置，跳过记忆库推送")
	}

	broadcastBackupStatus("ok", "序列 0 与今日记忆已妥善封存云端")
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
		}
	}

	// git commit（可能无变更，允许失败）
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = absDir
	if output, err := commitCmd.CombinedOutput(); err != nil {
		log.Printf("[Backup] git commit 输出: %s", string(output))
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
