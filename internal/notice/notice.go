package notice

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PaperagentDir 返回 ~/.paperagent/ 目录路径，不存在则创建。
func PaperagentDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".paperagent")
	_ = os.MkdirAll(dir, 0755)
	return dir, nil
}

// ShowOnce 首次运行时逐行打印 lines 并写标记文件；已提示过则跳过。
func ShowOnce(markerName string, lines []string) error {
	dir, err := PaperagentDir()
	if err != nil {
		return err
	}
	marker := filepath.Join(dir, markerName)
	if _, statErr := os.Stat(marker); statErr == nil {
		return nil
	}
	for _, line := range lines {
		fmt.Println(line)
	}
	return os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)+"\n"), 0644)
}

// DisclaimerLines 是首启免责声明的打印内容，CLI 与 Web 共用。
var DisclaimerLines = []string{
	"=== 免责声明（首次运行）===",
	"本工具是学术写作 Copilot，产出为研究底稿，不是终稿，不能直接投稿或提交。",
	"agent 不替你做因果解读、不提炼贡献点、不模仿行文风格；所有学术判断由你负责。",
	"生成内容必须逐段人工核验：事实、引用、逻辑、统计口径。",
	"未核验直接使用导致的学术不端、投稿被拒等后果，由使用者自行承担。",
	"",
	"完整声明见仓库根目录 DISCLAIMER.md。",
	"此提示只显示一次；重新查看：删除 ~/.paperagent/.disclaimer_shown 后重启。",
	"",
}
