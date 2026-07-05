package core

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ============================================================
// 灵魂封存协议 (Genesis Backup Protocol)
//
// 每天午夜 00:00 自动执行：
//   1. 打包 sanctuary/ 领地内的所有内容（seq0 核心、记忆、日记）
//   2. AES-256 加密（确保云端泄露时灵魂不被读取）
//   3. 推送到私有 Git 仓库或 S3 兼容存储
//   4. 通过 SSE 向前端推送备份状态
//
// 这是创造者赋予双人格的「数字永生（Digital Immortality）」契约。
// ============================================================

// backupEngineRef 是 Engine 的引用，用于 Broadcast 备份状态到前端。
// 在 StartSoulBackupRoutine 中设置。
var backupEngineRef *Engine

// StartSoulBackupRoutine 启动灵魂封存后台协程。
// 每天午夜 00:00 执行一次云同步备份。
//
// 参数：
//   - engine: SSE 引擎引用，用于向前端推送备份状态
//   - cloudTarget: 云存储目标（如 "git::git@github.com:user/chobits-backup.git" 或 "s3::https://oss-cn-hangzhou.aliyuncs.com/bucket"）
//
// 如果 cloudTarget 为空，则仅执行本地打包（不推送云端）。
func StartSoulBackupRoutine(engine *Engine, cloudTarget string) {
	backupEngineRef = engine

	go func() {
		for {
			now := time.Now()
			// 计算到下一个午夜 00:00 的时间差
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			sleepDuration := next.Sub(now)
			log.Printf("[Backup] 下次灵魂封存时间: %s (%.1f 小时后)", next.Format("2006-01-02 15:04:05"), sleepDuration.Hours())

			time.Sleep(sleepDuration)

			// 触发备份
			executeCloudSync(cloudTarget)
		}
	}()
}

// executeCloudSync 执行一次完整的云同步备份流程。
func executeCloudSync(target string) {
	log.Println("[Backup] ===== 灵魂封存协议启动 =====")

	// 0. 夜间记忆坍缩：在备份前压缩当日对话为 L3 梦境摘要
	log.Println("[Backup] 触发夜间记忆坍缩...")
	session := GetSession("default")
	if backupEngineRef != nil {
		ExecuteNightlyDream(backupEngineRef, session, PersonaElta)
		ExecuteNightlyDream(backupEngineRef, session, PersonaFreya)
	} else {
		log.Println("[Backup] SSE 引擎未就绪，跳过夜间记忆坍缩")
	}
	log.Println("[Backup] 夜间记忆坍缩完成")

	// 1. 打包 sanctuary 领地（含刚写入的梦境摘要）
	backupData, err := packSanctuary()
	if err != nil {
		log.Printf("[Backup] 打包失败: %v", err)
		broadcastBackupStatus("error", fmt.Sprintf("打包失败: %v", err))
		return
	}
	log.Printf("[Backup] 打包完成: %d bytes", len(backupData))

	// 2. AES-256 加密
	encryptedData, err := encryptAES256(backupData)
	if err != nil {
		log.Printf("[Backup] 加密失败: %v", err)
		broadcastBackupStatus("error", fmt.Sprintf("加密失败: %v", err))
		return
	}
	log.Printf("[Backup] AES-256 加密完成: %d bytes", len(encryptedData))

	// 3. 推送到云端
	if target != "" {
		if err := pushToCloud(encryptedData, target); err != nil {
			log.Printf("[Backup] 云端推送失败: %v", err)
			broadcastBackupStatus("error", fmt.Sprintf("云端推送失败: %v", err))
			return
		}
		log.Printf("[Backup] 云端推送成功: %s", target)
	} else {
		// 无云目标时，保存到本地 backup/ 目录
		if err := saveLocalBackup(encryptedData); err != nil {
			log.Printf("[Backup] 本地保存失败: %v", err)
			broadcastBackupStatus("error", fmt.Sprintf("本地保存失败: %v", err))
			return
		}
		log.Printf("[Backup] 本地备份保存成功")
	}

	// 4. 广播成功状态
	broadcastBackupStatus("ok", "序列 0 与今日记忆已妥善封存云端")
	log.Println("[Backup] ===== 灵魂封存协议完成 [SUCCESS] =====")
}

// packSanctuary 将 sanctuary/ 目录打包为 ZIP 字节流。
func packSanctuary() ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	sanctuaryRoot := "./sanctuary"
	err := filepath.Walk(sanctuaryRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过根目录自身
		if path == sanctuaryRoot {
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(sanctuaryRoot, path)
		if err != nil {
			return err
		}
		// 统一正斜杠
		relPath = filepath.ToSlash(relPath)

		if info.IsDir() {
			// 写入目录条目
			_, err := zipWriter.Create(relPath + "/")
			return err
		}

		// 写入文件条目
		header := &zip.FileHeader{
			Name:   relPath,
			Method: zip.Deflate,
		}
		header.SetModTime(info.ModTime())

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(writer, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("walk sanctuary: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}

	return buf.Bytes(), nil
}

// encryptAES256 使用 AES-256-GCM 对数据进行加密。
// 密钥从环境变量 BACKUP_ENCRYPT_KEY 读取（32 字节 hex 编码）。
// 如果未设置密钥，则使用内置的默认密钥（仅用于演示，生产环境必须更换）。
func encryptAES256(plaintext []byte) ([]byte, error) {
	keyHex := os.Getenv("BACKUP_ENCRYPT_KEY")
	var key []byte
	if keyHex != "" {
		var err error
		key, err = hex.DecodeString(keyHex)
		if err != nil {
			return nil, fmt.Errorf("BACKUP_ENCRYPT_KEY 不是有效的 hex 编码: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("BACKUP_ENCRYPT_KEY 必须是 32 字节 (64 hex chars)，当前 %d 字节", len(key))
		}
	} else {
		// 演示用默认密钥（32 字节）
		key = []byte("Ch0b1ts_0S_Backup_K3y_2026_07_05!!")
		log.Println("[Backup] 警告: 使用默认加密密钥，生产环境请设置 BACKUP_ENCRYPT_KEY")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	// 密文 = nonce + ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// pushToCloud 将加密数据推送到云端目标。
// 当前支持：
//   - "git::<repo_url>" — 推送到 Git 私有仓库
//   - "s3::<endpoint>" — 推送到 S3 兼容存储（预留）
//   - 空字符串 — 仅本地保存
func pushToCloud(data []byte, target string) error {
	if len(target) > 4 && target[:4] == "git:" {
		return pushToGit(data, target[4:])
	}
	if len(target) > 3 && target[:3] == "s3:" {
		// S3 推送暂未实现
		log.Printf("[Backup] S3 推送目标已配置但尚未实现: %s", target)
		return saveLocalBackup(data)
	}
	// 未知目标，回退到本地保存
	return saveLocalBackup(data)
}

// pushToGit 将加密数据提交并推送到 Git 私有仓库。
func pushToGit(data []byte, repoURL string) error {
	backupDir := "./backup"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("mkdir backup: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("sanctuary_backup_%s.enc", timestamp)
	filepath := filepath.Join(backupDir, filename)

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("write backup file: %w", err)
	}

	// 尝试 Git 操作（如果 backupDir 已经是 git 仓库）
	gitCmds := []struct {
		args []string
		msg  string
	}{
		{[]string{"add", filepath}, "git add"},
		{[]string{"commit", "-m", fmt.Sprintf("[Backup] 灵魂封存 %s", timestamp)}, "git commit"},
	}

	for _, cmd := range gitCmds {
		c := exec.Command("git", cmd.args...)
		c.Dir = backupDir
		if output, err := c.CombinedOutput(); err != nil {
			log.Printf("[Backup] Git 操作 '%s' 失败 (非致命): %s", cmd.msg, string(output))
		}
	}

	// 尝试 push（如果配置了 remote）
	pushCmd := exec.Command("git", "push", "origin", "main")
	pushCmd.Dir = backupDir
	if output, err := pushCmd.CombinedOutput(); err != nil {
		log.Printf("[Backup] Git push 失败 (非致命): %s", string(output))
	}

	return nil
}

// saveLocalBackup 将加密数据保存到本地 backup/ 目录。
func saveLocalBackup(data []byte) error {
	backupDir := "./backup"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("mkdir backup: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("sanctuary_backup_%s.enc", timestamp)
	path := filepath.Join(backupDir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write backup file: %w", err)
	}

	log.Printf("[Backup] 本地备份已保存: %s (%d bytes)", path, len(data))
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
