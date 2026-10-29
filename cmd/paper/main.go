package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"paperagent/internal/cite"
	"paperagent/internal/config"
	"paperagent/internal/design"
	"paperagent/internal/errhint"
	"paperagent/internal/export"
	"paperagent/internal/flow"
	"paperagent/internal/llm"
	"paperagent/internal/model"
	"paperagent/internal/pandocinstall"
	"paperagent/internal/pipeline"
	"paperagent/internal/quality"
	"paperagent/internal/render"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if hint := errhint.Explain(err); hint != "" {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "提示："+hint)
		}
		os.Exit(1)
	}
}

// hasAPIKey 检测是否配置了任一 LLM API Key。
func hasAPIKey() bool {
	if os.Getenv("DEEPSEEK_API_KEY") != "" {
		return true
	}
	if os.Getenv("ZAI_API_KEY") != "" {
		return true
	}
	if os.Getenv("ZHIPU_API_KEY") != "" {
		return true
	}
	return false
}

// paperagentDir 返回 ~/.paperagent/ 目录路径，不存在则创建。
func paperagentDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".paperagent")
	_ = os.MkdirAll(dir, 0755)
	return dir, nil
}

// printFirstRunGuide 首次运行引导：无 Key 时打印配置指引。
func printFirstRunGuide() {
	fmt.Println("=== 首次运行引导 ===")
	fmt.Println("未检测到 LLM API Key，当前使用 Mock 模式（生成占位内容，仅供流程测试）。")
	fmt.Println()
	fmt.Println("如需真实生成，请配置以下任一 Key：")
	fmt.Println("  1. DeepSeek（推荐，便宜且稳定）")
	fmt.Println("     申请：https://platform.deepseek.com/")
	fmt.Println("     配置：setx DEEPSEEK_API_KEY \"你的Key\"")
	fmt.Println("  2. 智谱（备用）")
	fmt.Println("     申请：https://open.bigmodel.cn/")
	fmt.Println("     配置：setx ZAI_API_KEY \"你的Key\"")
	fmt.Println()
	fmt.Println("配置后：关闭当前终端重新打开，或运行 . .\\load.ps1，再重新执行本命令。")
	fmt.Println("如果只想测试流程，可忽略此提示（Mock 模式不影响命令退出码）。")
	fmt.Println()
}

// printDataNotice C3：首次真实运行时告知数据外发。
// 已提示过则跳过；写标记文件 ~/.paperagent/.data_notice_shown。
func printDataNotice() {
	dir, err := paperagentDir()
	if err != nil {
		return
	}
	marker := filepath.Join(dir, ".data_notice_shown")
	if _, statErr := os.Stat(marker); statErr == nil {
		return
	}
	fmt.Println("=== 数据外发告知（首次运行）===")
	fmt.Println("本工具需要调用大模型 API 生成正文。以下内容会发送给 LLM 服务商（默认 DeepSeek）：")
	fmt.Println("  - 研究设计 JSON 中的主题、变量、模型、数据来源等描述")
	fmt.Println("  - 参考文献的标题、作者、年份、摘要、结论字段")
	fmt.Println("  - 章节标题与要点")
	fmt.Println("不会发送：CSMAR 原始数据文件、draft.md 已有正文、API Key")
	fmt.Println()
	fmt.Println("如有未发表数据或敏感信息，请先脱敏再运行。")
	fmt.Println("此提示只显示一次；撤回同意：删除 ~/.paperagent/.data_notice_shown 后重启即可重新提示。")
	fmt.Println()
	_ = os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)+"\n"), 0644)
}

// estimateTokens C4：跑前粗估 token 用量和成本。
// 按 5 章节 × 3000 token + refs 数 × 50 token 估算，返回 (低估值, 高估值)。
func estimateTokens(refsCount int) (int64, int64) {
	base := int64(5*3000 + refsCount*50)
	low := base * 7 / 10
	high := base * 13 / 10
	return low, high
}

// estimateCost 按 DeepSeek deepseek-chat 保守定价估成本（元）：
// 输入 1 元/百万 token，输出 2 元/百万 token。仅作量级参考，以官网为准。
func estimateCost(tokens int64) float64 {
	// 假设 input:output ≈ 6:5（根据实测）
	input := tokens * 6 / 11
	output := tokens - input
	return float64(input)*1e-6 + float64(output)*2e-6
}

// printCostEstimate 打印成本预估。
func printCostEstimate(refsCount int) {
	low, high := estimateTokens(refsCount)
	costLow := estimateCost(low)
	costHigh := estimateCost(high)
	fmt.Printf("ESTIMATED_TOKENS=%d-%d\n", low, high)
	fmt.Printf("ESTIMATED_COST=%.4f-%.4f 元（按 deepseek-chat 定价，以官网为准）\n", costLow, costHigh)
}

// printPandocHint 导出前若未找到 pandoc，提示安装方法。
func printPandocHint() {
	if pandocinstall.IsInstalled() {
		return
	}
	if os.Getenv("PANDOC_PATH") != "" {
		return
	}
	fmt.Println("提示：未检测到 pandoc，--export 可能失败。")
	fmt.Println("      自动安装：paperagent --install-pandoc")
	fmt.Println("      手动安装：winget install JohnMacFarlane.Pandoc")
	fmt.Println()
}

// usageLogPath 返回 ~/.paperagent/usage.log 路径。
func usageLogPath() (string, error) {
	dir, err := paperagentDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "usage.log"), nil
}

// readTodayUsage 读 usage.log，累加今天的所有 total= 值。文件不存在返回 0。
func readTodayUsage() (int64, error) {
	path, err := usageLogPath()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	today := time.Now().Format("2006-01-02")
	var total int64
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, today) {
			continue
		}
		idx := strings.Index(line, "total=")
		if idx < 0 {
			continue
		}
		rest := line[idx+len("total="):]
		if sp := strings.IndexByte(rest, ' '); sp > 0 {
			rest = rest[:sp]
		}
		if n, err := strconv.ParseInt(rest, 10, 64); err == nil {
			total += n
		}
	}
	return total, nil
}

// reportUsage 从 client 读取用量，打印、写日志，返回本次 total。
func reportUsage(client llm.Client, mode string, elapsed time.Duration) int64 {
	u, ok := client.(llm.UsageReporter)
	if !ok {
		return 0
	}
	in, out := u.Usage()
	if in == 0 && out == 0 {
		return 0
	}
	total := in + out
	fmt.Printf("TOKENS_INPUT=%d\n", in)
	fmt.Printf("TOKENS_OUTPUT=%d\n", out)
	fmt.Printf("TOKENS_TOTAL=%d\n", total)

	path, err := usageLogPath()
	if err != nil {
		return total
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return total
	}
	defer f.Close()
	line := fmt.Sprintf("%s mode=%s input=%d output=%d total=%d elapsed=%.1fs\n",
		time.Now().Format("2006-01-02T15:04:05"), mode, in, out, total, elapsed.Seconds())
	_, _ = f.WriteString(line)
	return total
}

// historyDir 返回 ~/.paperagent/history/ 目录路径，不存在则创建。
func historyDir() (string, error) {
	dir, err := paperagentDir()
	if err != nil {
		return "", err
	}
	h := filepath.Join(dir, "history")
	_ = os.MkdirAll(h, 0755)
	return h, nil
}

// historyEntry 单次运行的元数据。
type historyEntry struct {
	Time         string `json:"time"`
	Topic        string `json:"topic"`
	Mode         string `json:"mode"`
	OutputFile   string `json:"output_file"`
	Tokens       int64  `json:"tokens"`
	Quality      string `json:"quality"`
	Placeholders int    `json:"placeholders"`
}

// saveHistory 保存本次运行到 history 目录，保留最近 20 次。
func saveHistory(topic, mode, quality string, placeholders int, tokens int64, mdContent string) {
	hdir, err := historyDir()
	if err != nil {
		return
	}
	ts := time.Now().Format("20060102-150405")
	entryDir := filepath.Join(hdir, ts)
	if err := os.MkdirAll(entryDir, 0755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(entryDir, "draft.md"), []byte(mdContent), 0644)
	meta := historyEntry{
		Time:         time.Now().Format(time.RFC3339),
		Topic:        topic,
		Mode:         mode,
		OutputFile:   "draft.md",
		Tokens:       tokens,
		Quality:      quality,
		Placeholders: placeholders,
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(filepath.Join(entryDir, "meta.json"), metaJSON, 0644)
	pruneHistory(hdir, 20)
}

// pruneHistory 保留最近 n 个历史目录，删更旧的。
func pruneHistory(hdir string, n int) {
	entries, err := os.ReadDir(hdir)
	if err != nil || len(entries) <= n {
		return
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for i := 0; i < len(names)-n; i++ {
		_ = os.RemoveAll(filepath.Join(hdir, names[i]))
	}
}

// listHistory 打印最近 10 次历史记录。
func listHistory() error {
	hdir, err := historyDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(hdir)
	if err != nil {
		fmt.Println("无历史记录")
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		fmt.Println("无历史记录")
		return nil
	}
	sort.Strings(names)
	if len(names) > 10 {
		names = names[len(names)-10:]
	}
	fmt.Printf("最近 %d 次运行（#1=最近）：\n", len(names))
	for i := len(names) - 1; i >= 0; i-- {
		dir := filepath.Join(hdir, names[i])
		metaData, _ := os.ReadFile(filepath.Join(dir, "meta.json"))
		var m historyEntry
		_ = json.Unmarshal(metaData, &m)
		fmt.Printf("  #%-2d %s  mode=%s  topic=%s  tokens=%d  quality=%s  placeholders=%d\n",
			len(names)-i, m.Time, m.Mode, m.Topic, m.Tokens, m.Quality, m.Placeholders)
	}
	return nil
}

// showHistory 把第 n 次历史记录（1=最近）的 draft 复制到当前目录。
func showHistory(n int) error {
	hdir, err := historyDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(hdir)
	if err != nil {
		return fmt.Errorf("无历史记录")
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if n < 1 || n > len(names) {
		return fmt.Errorf("序号 %d 超出范围（1-%d）", n, len(names))
	}
	sort.Strings(names)
	dir := filepath.Join(hdir, names[len(names)-n])
	src := filepath.Join(dir, "draft.md")
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("读取历史 draft 失败: %w", err)
	}
	out := fmt.Sprintf("draft_history_%d.md", n)
	if err := os.WriteFile(out, data, 0644); err != nil {
		return fmt.Errorf("写出失败: %w", err)
	}
	fmt.Printf("已复制到: %s\n", out)
	return nil
}

func run(args []string) error {
	fs := flag.NewFlagSet("paper", flag.ContinueOnError)

	topicFlag := fs.String("topic", "", "论文主题（为空则用 step1 默认）")
	modeFlag := fs.String("mode", "thesis", "论文模式：thesis / journal")
	refsFlag := fs.String("refs", "", "参考文献 JSON 路径（可选）")
	designFlag := fs.String("design", "", "研究设计 JSON 路径（可选，补上研究设计/实证结果章节）")
	polishDesignFlag := fs.Bool("polish-design", false, "研究设计节先用 LLM 润色，失败回落模板（默认直接用模板）")
	outputFlag := fs.String("output", "draft.md", "输出 Markdown 路径")
	verifyFlag := fs.Bool("verify", false, "是否校验参考文献真实性")
	mailtoFlag := fs.String("mailto", "", "OpenAlex 礼貌池邮箱（可选）")
	exportFlag := fs.String("export", "", "导出格式：docx / tex / html（为空则只生成 md）")
	resumeFlag := fs.Bool("resume", true, "断点续跑：检测到 <output>.progress.json 时从断点继续，false 则忽略进度文件从头跑")
	installPandocFlag := fs.Bool("install-pandoc", false, "下载并安装 pandoc 到 ~/.paperagent/pandoc/，安装后退出")
	maxPerRunFlag := fs.Int64("max-tokens-per-run", 0, "单次运行 token 上限，0 表示不限（超限保留结果但打印警告）")
	maxPerDayFlag := fs.Int64("max-tokens-per-day", 0, "单日累计 token 上限，0 表示不限（超限直接拒绝运行）")
	noEstimateFlag := fs.Bool("no-estimate", false, "跳过跑前成本预估打印")
	historyFlag := fs.Bool("history", false, "列出最近 10 次历史运行记录")
	historyShowFlag := fs.Int("history-show", 0, "把第 N 次历史记录（1=最近）的 draft 复制到当前目录")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// 独立子命令：--install-pandoc 只装 pandoc，不生成论文。
	if *installPandocFlag {
		fmt.Println("=== 安装 pandoc ===")
		bin, err := pandocinstall.Install(context.Background(), "", func(msg string) {
			fmt.Println(msg)
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "自动下载失败，推荐以下手动方案（任一即可）：")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "方案 A（推荐，最快）：winget install JohnMacFarlane.Pandoc")
			fmt.Fprintln(os.Stderr, "  安装后运行：cd E:\\lunwenagent; . .\\load.ps1 即可自动识别")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "方案 B：从 https://pandoc.org/installing.html 下载 MSI 安装包")
			fmt.Fprintln(os.Stderr, "  安装后设置环境变量：setx PANDOC_PATH \"C:\\Program Files\\Pandoc\\pandoc.exe\"")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "方案 C：向项目维护者索取离线 pandoc.exe，放入 ~/.paperagent/pandoc/ 即可")
			return fmt.Errorf("安装 pandoc 失败: %w", err)
		}
		fmt.Printf("pandoc 已安装：%s\n", bin)
		fmt.Println("后续导出无需再手动配置 PANDOC_PATH。")
		return nil
	}

	// D3：独立子命令 --history 和 --history-show
	if *historyFlag {
		return listHistory()
	}
	if *historyShowFlag > 0 {
		return showHistory(*historyShowFlag)
	}

	// B3：首次运行引导。只在真实 LLM 模式下提示，Mock 测试不打印。
	if os.Getenv("LLM_PROVIDER") != "mock" && !hasAPIKey() {
		printFirstRunGuide()
	}

	// C3：数据外发告知。只在真实 LLM 模式下提示，Mock 测试不打印。
	if os.Getenv("LLM_PROVIDER") != "mock" && hasAPIKey() {
		printDataNotice()
	}

	// 导出前提示 pandoc 缺失，不阻塞生成。
	if *exportFlag != "" {
		printPandocHint()
	}

	cfg := config.Config{Mock: true}
	cfg.Weights = config.DefaultWeights()
	cfg.Mode = model.Mode(*modeFlag)
	cfg.PolishDesign = *polishDesignFlag
	cfg.MaxTokensPerRun = *maxPerRunFlag
	cfg.MaxTokensPerDay = *maxPerDayFlag
	if *resumeFlag {
		cfg.ProgressPath = *outputFlag + ".progress.json"
	}

	switch cfg.Mode {
	case model.ModeThesis, model.ModeJournal:
	default:
		return fmt.Errorf("invalid --mode %q, must be thesis or journal", cfg.Mode)
	}

	// C4：跑前成本预估，不阻塞。只在真实 LLM 模式下打印。
	refsCount := 0
	if *refsFlag != "" {
		// 先数一下 refs 条数用于估算，正式加载在后面
		if rs, err := cite.LoadRefs(*refsFlag); err == nil {
			refsCount = len(rs)
		}
	}
	if !*noEstimateFlag && os.Getenv("LLM_PROVIDER") != "mock" && hasAPIKey() {
		printCostEstimate(refsCount)
	}

	// C2：跑前检查单日累计用量。
	if cfg.MaxTokensPerDay > 0 {
		today, err := readTodayUsage()
		if err == nil {
			fmt.Printf("TODAY_TOKENS=%d (limit=%d)\n", today, cfg.MaxTokensPerDay)
			if today >= cfg.MaxTokensPerDay {
				return fmt.Errorf("今日 token 用量已达上限（%d / %d）。请明日再跑，或提高 --max-tokens-per-day（0 表示不限）", today, cfg.MaxTokensPerDay)
			}
		}
	}

	if *designFlag != "" {
		d, err := design.Load(*designFlag)
		if err != nil {
			return fmt.Errorf("load design failed: %w", err)
		}
		cfg.Design = &d
		fmt.Printf("design loaded: %s\n", *designFlag)
	}

	if *refsFlag != "" {
		refs, err := cite.LoadRefs(*refsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load refs for step3 failed (continuing without refs): %v\n", err)
		} else {
			cfg.Refs = refs
			fmt.Printf("refs loaded for step3: %d refs\n", len(refs))
		}
	}

	start := time.Now()
	orch := flow.New(cfg, nil)

	input := map[string]interface{}{}
	if *topicFlag != "" {
		input["topic"] = *topicFlag
	}
	if cfg.Design != nil && cfg.Design.Topic != "" && *topicFlag == "" {
		input["topic"] = cfg.Design.Topic
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshal input failed: %w", err)
	}

	step1JSON, err := orch.Run(inputJSON)
	if err != nil {
		return fmt.Errorf("step1 failed: %w", err)
	}

	ctx := pipeline.NewContext(cfg.Mock, cfg.Strict)

	st2 := pipeline.Registry["step2_frame"]
	if st2 == nil {
		return fmt.Errorf("step2_frame not registered")
	}
	step2JSON, err := st2.Run(ctx, step1JSON)
	if err != nil {
		return fmt.Errorf("step2 failed: %w", err)
	}

	st3 := pipeline.Registry["step3_litreview"]
	if st3 == nil {
		return fmt.Errorf("step3_litreview not registered")
	}
	step3JSON, err := st3.Run(ctx, step2JSON)
	if err != nil {
		return fmt.Errorf("step3 failed: %w", err)
	}

	var frame model.Frame
	if err := json.Unmarshal(step3JSON, &frame); err != nil {
		return fmt.Errorf("parse frame failed: %w", err)
	}

	// D2：翻译 RiskHint 里的技术错误
	for i, h := range frame.RiskHints {
		if hint := errhint.Explain(errors.New(h)); hint != "" {
			frame.RiskHints[i] = h + "\n    提示：" + hint
		}
	}

	var refsText string
	if *refsFlag != "" {
		refsText, err = cite.ProcessRefs(*refsFlag, *verifyFlag, *mailtoFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "refs failed (continuing without refs): %v\n", err)
		}
	}

	md := render.Markdown(frame, refsText)

	if err := os.WriteFile(*outputFlag, []byte(md), 0644); err != nil {
		return fmt.Errorf("write output failed: %w", err)
	}
	fmt.Printf("written: %s\n", *outputFlag)

	report := quality.Check(md, cfg.Refs)
	report.Print()
	qualityStr := "FAIL"
	if report.AllPass() {
		qualityStr = "PASS"
		fmt.Println("QUALITY_STATUS=PASS")
	} else {
		fmt.Println("QUALITY_STATUS=FAIL")
	}
	fmt.Printf("QUALITY_PLACEHOLDERS=%d\n", report.PlaceholderCount)

	// C1：打印 token 用量并写日志
	total := reportUsage(orch.LLM, string(cfg.Mode), time.Since(start))

	// C2：跑后检查单次上限，超限保留结果但打印警告
	if cfg.MaxTokensPerRun > 0 && total > cfg.MaxTokensPerRun {
		fmt.Fprintf(os.Stderr, "警告：本次 token 用量 %d 超过单次上限 %d，已保留结果但建议检查 prompt 是否过长\n", total, cfg.MaxTokensPerRun)
		fmt.Println("TOKENS_OVER_LIMIT=1")
	}

	// D3：保存历史记录（保留最近 20 次）
	saveHistory(frame.Title, string(cfg.Mode), qualityStr, report.PlaceholderCount, total, md)

	if *exportFlag == "" {
		return nil
	}

	outPath := replaceExt(*outputFlag, "."+*exportFlag)
	e := export.NewPandocExporter()
	if err := e.Export(context.Background(), *outputFlag, outPath); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}
	fi, err := os.Stat(outPath)
	if err != nil {
		return fmt.Errorf("export output missing: %w", err)
	}
	fmt.Printf("exported: %s (%d bytes)\n", outPath, fi.Size())
	return nil
}

func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + newExt
	}
	return path[:len(path)-len(ext)] + newExt
}
