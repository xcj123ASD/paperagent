// Package validate 提供输入文件的安全检查。
package validate

import (
	"fmt"
	"os"
)

// DefaultMaxBytes 默认文件大小上限：10MB。
// refs_real.json 50 篇约 44KB，10MB 是 200 倍余量，可容纳 1000+ 篇。
const DefaultMaxBytes int64 = 10 << 20

// FileSize 检查文件大小，超过 limitBytes 报错。
// limitBytes <= 0 时用 DefaultMaxBytes。
func FileSize(path string, limitBytes int64) error {
	if limitBytes <= 0 {
		limitBytes = DefaultMaxBytes
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("读取文件信息失败: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("路径是目录而非文件: %s", path)
	}
	if info.Size() > limitBytes {
		return fmt.Errorf("文件过大: %d 字节（上限 %d 字节，约 %d MB）",
			info.Size(), limitBytes, limitBytes/1024/1024)
	}
	return nil
}
