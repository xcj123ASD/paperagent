package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"paperagent/internal/cite"
	"paperagent/internal/config"
	"paperagent/internal/design"
	"paperagent/internal/export"
	"paperagent/internal/flow"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
	"paperagent/internal/render"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg := config.Config{Mock: true}
	cfg.Weights = config.DefaultWeights()
	cfg.Mode = model.Mode(*modeFlag)
	cfg.PolishDesign = *polishDesignFlag

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