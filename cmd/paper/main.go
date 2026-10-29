package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"paperagent/internal/cite"
	"paperagent/internal/config"
	"paperagent/internal/design"
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

// reportUsage 从 client 读取用量，打印并追加写 ~/.paperagent/usage.log。
func reportUsage(client llm.Client, mode string, elapsed time.Duration) {
	u, ok := client.(llm.UsageReporter)
	if !ok {
		return
	}
	in, out := u.Usage()
	if in == 0 && out == 0 {
		return
	}
	total := in + out
	fmt.Printf("TOKENS_INPUT=%d\n", in)
	fmt.Printf("TOKENS_OUTPUT=%d\n", out)
	fmt.Printf("TOKENS_TOTAL=%d\n", total)

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(home, ".paperagent")
	_ = os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "usage.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	line := fmt.Sprintf("%s mode=%s input=%d output=%d total=%d elapsed=%.1fs\n",
		time.Now().Format("2006-01-02T15:04:05"), mode, in, out, total, elapsed.Seconds())
	_, _ = f.WriteString(line)
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

	// B3：首次运行引导。只在真实 LLM 模式下提示，Mock 测试不打印。
	if os.Getenv("LLM_PROVIDER") != "mock" && !hasAPIKey() {
		printFirstRunGuide()
	}

	// 导出前提示 pandoc 缺失，不阻塞生成。
	if *exportFlag != "" {
		printPandocHint()
	}

	cfg := config.Config{Mock: true}
	cfg.Weights = config.DefaultWeights()
	cfg.Mode = model.Mode(*modeFlag)
	cfg.PolishDesign = *polishDesignFlag
	if *resumeFlag {
		cfg.ProgressPath = *outputFlag + ".progress.json"
	}

	switch cfg.Mode {
	case model.ModeThesis, model.ModeJournal:
	default:
		return fmt.Errorf("invalid --mode %q, must be thesis or journal", cfg.Mode)
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
	if report.AllPass() {
		fmt.Println("QUALITY_STATUS=PASS")
	} else {
		fmt.Println("QUALITY_STATUS=FAIL")
	}
	fmt.Printf("QUALITY_PLACEHOLDERS=%d\n", report.PlaceholderCount)

	// C1：打印 token 用量并写日志
	reportUsage(orch.LLM, string(cfg.Mode), time.Since(start))

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
