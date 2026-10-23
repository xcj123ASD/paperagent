package stata

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// 默认 Stata 路径：本机实测路径。
// 可通过环境变量 STATA_PATH 覆盖，便于换机器或换版本。
const defaultStataPath = `C:\Program Files\Stata18\StataMP-64.exe`

// 默认超时：Stata 批处理跑大回归可能较慢，给 5 分钟。
const defaultTimeout = 5 * time.Minute

// Runner 负责调用本地 Stata 执行 .do 文件。
//
// 安全设计：
//   - 只执行本地 Stata，绝不做服务端
//   - 参数数组传 exec.Command，不走 shell 字符串拼接，防注入
//   - 由调用方显式调用 Run()，runner 本身不决定是否执行
type Runner struct {
	StataPath string        // Stata 可执行文件路径
	Timeout   time.Duration // 单次执行超时
}

// NewRunner 构造 Runner，自动查找 Stata 路径。
//
// 查找顺序：
//  1. 环境变量 STATA_PATH
//  2. defaultStataPath（本机实测路径）
//
// 都找不到时返回 error，调用方应提示用户设置 STATA_PATH 或安装 Stata。
func NewRunner() (*Runner, error) {
	path, err := findStata()
	if err != nil {
		return nil, err
	}
	return &Runner{
		StataPath: path,
		Timeout:   defaultTimeout,
	}, nil
}

// findStata 按查找顺序定位 Stata 可执行文件。
func findStata() (string, error) {
	if envPath := os.Getenv("STATA_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", fmt.Errorf("stata: STATA_PATH 指向的文件不存在: %s", envPath)
	}

	if _, err := os.Stat(defaultStataPath); err == nil {
		return defaultStataPath, nil
	}

	return "", fmt.Errorf("stata: 未找到 Stata，请设置环境变量 STATA_PATH 指向 Stata 可执行文件")
}

// IsAvailable 检查 Stata 是否可用（文件存在）。
// 供调用方在提示用户前判断，不触发执行。
func (r *Runner) IsAvailable() bool {
	if r == nil || r.StataPath == "" {
		return false
	}
	_, err := os.Stat(r.StataPath)
	return err == nil
}

// Run 执行指定的 .do 文件。
//
// 使用 Stata 批处理模式：`StataMP-64.exe -b do "file.do"`。
//   - `-b` 静默模式（batch），不弹 GUI
//   - 参数分数组传，不走 shell 拼接，命令注入无法通过参数生效
//
// 返回 stdout+stderr 合并的输出，供调用方判断成败或记日志。
// 调用方应在执行前先跑 CheckSafety(doText) 做静态检查。
//
// 注意：本方法不检查 .do 文件内容，只负责执行。安全检查是独立的一层。
func (r *Runner) Run(ctx context.Context, doPath string) (string, error) {
	if r == nil || r.StataPath == "" {
		return "", fmt.Errorf("stata: Runner 未初始化或 StataPath 为空")
	}
	if _, err := os.Stat(doPath); err != nil {
		return "", fmt.Errorf("stata: .do 文件不存在: %s: %w", doPath, err)
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 参数数组：StataMP-64.exe -b do "xxx.do"
	// 不走 shell，路径里的特殊字符不会被解释
	cmd := exec.CommandContext(runCtx, r.StataPath, "-b", "do", doPath)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	output := buf.String()

	if runCtx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("stata: 执行超时（%s）: %w", timeout, runCtx.Err())
	}
	if err != nil {
		return output, fmt.Errorf("stata: 执行失败: %w\n输出:\n%s", err, output)
	}
	return output, nil
}
