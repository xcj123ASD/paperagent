package stata

import (
	"fmt"
	"strings"
)

// dangerousPrefixes 行首危险 token。
//
// 覆盖范围比文档任务 7 的最小要求（!cmd / shell / !del）更广：
//   - `!`         Stata 系统命令前缀，任何行首 `!` 都会执行系统命令
//   - `shell`     等同 `!`，执行系统 shell
//   - `rm`/`erase`/`del`  删文件，可能删用户原始 .dta
//   - `winexec`   启动外部程序
//   - `copy`/`move`  复制/移动文件，可能覆盖用户数据
//
// 末尾带空格的（"rm " / "del "）防止误伤 "rmse" 这类合法变量名。
var dangerousPrefixes = []string{
	"!",
	"shell",
	"rm ",
	"rm\t",
	"erase ",
	"erase\t",
	"del ",
	"del\t",
	"winexec",
	"copy ",
	"copy\t",
	"move ",
	"move\t",
}

// CheckSafety 扫描 .do 文本，拦截可能执行系统命令或删文件的危险行。
//
// 逐行扫描，跳过空行与整行注释（以 * 开头），避免模板里的说明文字误报。
// 只检查 trim 后的行首 token，不做字符串内匹配——这比 strings.Contains
// 更保守，宁可漏报字符串内的命令片段，也不误报合法正文。
func CheckSafety(doText string) error {
	lines := strings.Split(doText, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "*") {
			continue
		}
		lower := strings.ToLower(trimmed)
		for _, p := range dangerousPrefixes {
			if strings.HasPrefix(lower, p) {
				return fmt.Errorf("stata: 第 %d 行含危险命令 %q，已拦截: %s", i+1, p, line)
			}
		}
	}
	return nil
}
