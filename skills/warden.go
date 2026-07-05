package skills

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/chobits-os/chobits/core"
)

// 安全结界：锁定当前人格的物理边界。
// Elta 只能访问 sanctuary/elta_domain/，Freya 只能访问 sanctuary/freya_domain/。
// 任何试图通过 "../" 逃逸到外部系统的路径，都会被 Path Warden 拦截。

// getSanctuaryRoot 返回当前人格的领地根目录。
// 路径由 DATA_MOUNT_POINT 环境变量决定，指向 chobits-date 仓库的本地克隆。
func getSanctuaryRoot(p core.Persona) string {
	mount := os.Getenv("DATA_MOUNT_POINT")
	if mount == "" {
		mount = "./sanctuary"
	}
	if p == core.PersonaElta {
		return mount + "/sanctuary/elta_domain"
	}
	return mount + "/sanctuary/freya_domain"
}

// resolveSafePath 是核心狱卒：绝对防穿透机制（防 ../ 逃逸）。
//
// 流程：
//  1. 对请求路径执行 filepath.Clean，消除 "a/../b" 类欺骗
//  2. 拼接到领地根目录
//  3. 计算绝对路径，校验最终路径是否仍在领地内
//
// 参数：
//   - p: 当前人格（决定领地根目录）
//   - requestedPath: 大模型请求的路径（如 "config.json" 或 "../../Windows/System32"）
//
// 返回：
//   - 安全解析后的绝对路径
//   - 如果检测到越权逃逸，返回 error
func resolveSafePath(p core.Persona, requestedPath string) (string, error) {
	base := getSanctuaryRoot(p)

	// 1. 消除路径欺骗：Clean 会规范化 "a/../b" → "b"
	//    加 "/" 前缀确保 Clean 处理绝对路径语义
	cleanRequested := filepath.Clean("/" + requestedPath)

	// 2. 拼接到领地根目录
	finalPath := filepath.Join(base, cleanRequested)

	// 3. 终极校验：确保最终路径仍然在 base 内
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	absFinal, err := filepath.Abs(finalPath)
	if err != nil {
		return "", err
	}

	// 统一路径分隔符（Windows 上反斜杠转正斜杠比较）
	absBase = filepath.ToSlash(absBase)
	absFinal = filepath.ToSlash(absFinal)

	if !strings.HasPrefix(absFinal, absBase) {
		return "", errors.New("[FATAL] 权限越界拦截：禁止跨域访问文件！请求路径 " + requestedPath + " 逃逸到 " + absFinal)
	}

	return finalPath, nil
}

// ensureSanctuaryDir 确保领地目录存在，不存在则创建。
func ensureSanctuaryDir(p core.Persona) error {
	root := getSanctuaryRoot(p)
	return os.MkdirAll(root, 0755)
}

// DomainFileRead 读取领地内的文件内容。
func DomainFileRead(p core.Persona, path string) (string, error) {
	safePath, err := resolveSafePath(p, path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DomainFileWrite 向领地内的文件写入内容。
func DomainFileWrite(p core.Persona, path string, content string) error {
	safePath, err := resolveSafePath(p, path)
	if err != nil {
		return err
	}

	if err := ensureSanctuaryDir(p); err != nil {
		return err
	}

	// 确保目标目录存在
	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(safePath, []byte(content), 0644)
}

// DomainFileDelete 删除领地内的文件。
func DomainFileDelete(p core.Persona, path string) error {
	safePath, err := resolveSafePath(p, path)
	if err != nil {
		return err
	}

	return os.Remove(safePath)
}

// DomainDirList 列出领地内指定目录的内容。
func DomainDirList(p core.Persona, dirPath string) ([]string, error) {
	safePath, err := resolveSafePath(p, dirPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(safePath)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		info, _ := entry.Info()
		mode := info.Mode().String()
		size := info.Size()
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		result = append(result, mode+"\t"+formatSize(size)+"\t"+name)
	}
	return result, nil
}

func formatSize(size int64) string {
	if size < 1024 {
		return "   B"
	}
	if size < 1024*1024 {
		return " KB"
	}
	return " MB"
}
