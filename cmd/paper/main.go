package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"paperagent/internal/cite"
	"paperagent/internal/config"
	"paperagent/internal/design"
	"paperagent/internal/export"
	"paperagent/internal/flow"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
)

func main() {
	topicFlag := flag.String("topic", "", "论文主题（为空则用 step1 默认）")
	modeFlag := flag.String("mode", "thesis", "论文模式：thesis / journal")
	refsFlag := flag.String("refs", "", "参考文献 JSON 路径（可选）")
	designFlag := flag.String("design", "", "研究设计 JSON 路径（可选，补上研究设计/实证结果章节）")
	polishDesignFlag := flag.Bool("polish-design", false, "研究设计节先用 LLM 润色，失败回落模板（默认直接用模板）")
	outputFlag := flag.String("output", "draft.md", "输出 Markdown 路径")
	verifyFlag := flag.Bool("verify", false, "是否校验参考文献真实性")
	mailtoFlag := flag.String("mailto", "", "OpenAlex 礼貌池邮箱（可选）")
	exportFlag := flag.String("export", "", "导出格式：docx / tex / html（为空则只生成 md）")
	flag.Parse()

	cfg := config.Config{Mock: true}
	cfg.Weights = config.DefaultWeights()
	cfg.Mode = model.Mode(*modeFlag)

	switch cfg.Mode {
	case model.ModeThesis, model.ModeJournal:
	default:
		fmt.Fprintf(os.Stderr, "invalid --mode %q, must be thesis or journal\n", cfg.Mode)
		os.Exit(1)
	}

	if *designFlag != "" {
		d, err := design.Load(*designFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load design failed: %v\n", err)
			os.Exit(1)
		}
		cfg.Design = &d
		fmt.Printf("design loaded: %s\n", *designFlag)
	}
	cfg.PolishDesign = *polishDesignFlag

	orch := flow.New(cfg, nil)

	input := map[string]interface{}{}
	if *topicFlag != "" {
		input["topic"] = *topicFlag
	}
	if cfg.Design != nil && cfg.Design.Topic != "" && *topicFlag == "" {
		input["topic"] = cfg.Design.Topic
	}
	inputJSON, _ := json.Marshal(input)

	step1JSON, err := orch.Run(inputJSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "step1 failed: %v\n", err)
		os.Exit(1)
	}

	ctx := pipeline.NewContext(cfg.Mock, cfg.Strict)

	st2 := pipeline.Registry["step2_frame"]
	if st2 == nil {
		fmt.Fprintln(os.Stderr, "step2_frame not registered")
		os.Exit(1)
	}
	step2JSON, err := st2.Run(ctx, step1JSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "step2 failed: %v\n", err)
		os.Exit(1)
	}

	st3 := pipeline.Registry["step3_litreview"]
	if st3 == nil {
		fmt.Fprintln(os.Stderr, "step3_litreview not registered")
		os.Exit(1)
	}
	step3JSON, err := st3.Run(ctx, step2JSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "step3 failed: %v\n", err)
		os.Exit(1)
	}

	var frame model.Frame
	if err := json.Unmarshal(step3JSON, &frame); err != nil {
		fmt.Fprintf(os.Stderr, "parse frame failed: %v\n", err)
		os.Exit(1)
	}

	var refsText string
	if *refsFlag != "" {
		refsText, err = processRefs(*refsFlag, *verifyFlag, *mailtoFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "refs failed (continuing without refs): %v\n", err)
		}
	}

	md := renderMarkdown(frame, refsText)

	if err := os.WriteFile(*outputFlag, []byte(md), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write output failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("written: %s\n", *outputFlag)

	if *exportFlag == "" {
		return
	}

	outPath := replaceExt(*outputFlag, "."+*exportFlag)
	e := export.NewPandocExporter()
	if err := e.Export(context.Background(), *outputFlag, outPath); err != nil {
		fmt.Fprintf(os.Stderr, "export failed: %v\n", err)
		os.Exit(1)
	}
	fi, err := os.Stat(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "export output missing: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("exported: %s (%d bytes)\n", outPath, fi.Size())
}

func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + newExt
	}
	return path[:len(path)-len(ext)] + newExt
}

func processRefs(path string, verify bool, mailto string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var refs []cite.Reference
	if err := json.Unmarshal(data, &refs); err != nil {
		return "", err
	}
	if len(refs) == 0 {
		return "", nil
	}
	if verify {
		v := cite.NewOpenAlexVerifier(mailto)
		for i := range refs {
			res, err := v.Verify(context.Background(), refs[i])
			if err != nil {
				continue
			}
			if res.Status == cite.VerifyFound && res.Matched != nil && res.Confidence >= 0.8 {
				mergeRef(&refs[i], *res.Matched)
			}
		}
	}
	f := cite.GBT7714{}
	return f.Format(refs)
}

func mergeRef(dst *cite.Reference, src cite.Reference) {
	if dst.Title == "" {
		dst.Title = src.Title
	}
	if dst.Year == 0 {
		dst.Year = src.Year
	}
	if dst.Container == "" {
		dst.Container = src.Container
	}
	if dst.Volume == "" {
		dst.Volume = src.Volume
	}
	if dst.Issue == "" {
		dst.Issue = src.Issue
	}
	if dst.Pages == "" {
		dst.Pages = src.Pages
	}
	if dst.DOI == "" {
		dst.DOI = src.DOI
	}
	if len(dst.Authors) == 0 {
		dst.Authors = src.Authors
	}
	if dst.Lang == "" {
		dst.Lang = src.Lang
	}
	if dst.Type == "" {
		dst.Type = src.Type
	}
}

func renderMarkdown(f model.Frame, refsText string) string {
	var sb strings.Builder
	sb.WriteString("# " + f.Title + "\n\n")
	if len(f.Keywords) > 0 {
		sb.WriteString("**关键词：** " + strings.Join(f.Keywords, "；") + "\n\n")
	}
	if f.AbstractCN != "" {
		sb.WriteString("## 摘要\n\n" + f.AbstractCN + "\n\n")
	}
	for _, sec := range f.Sections {
		sb.WriteString("## " + sec.Name + "\n\n")
		if sec.Content != "" {
			sb.WriteString(sec.Content + "\n\n")
		} else {
			sb.WriteString("*（本节未生成）*\n\n")
		}
	}
	if refsText != "" {
		sb.WriteString("## 参考文献\n\n")
		for _, line := range strings.Split(refsText, "\n") {
			if strings.TrimSpace(line) != "" {
				sb.WriteString(line + "\n\n")
			}
		}
	}
	if len(f.RiskHints) > 0 {
		sb.WriteString("---\n\n**风险提示**\n\n")
		for _, h := range f.RiskHints {
			sb.WriteString("- " + h + "\n")
		}
	}
	return sb.String()
}