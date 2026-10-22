package main

import (
	"flag"
	"fmt"
	"os"

	"paperagent/internal/config"
	"paperagent/internal/flow"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
)

func main() {
	modeFlag := flag.String("mode", "thesis", "论文模式：thesis 或 journal")
	flag.Parse()

	cfg := config.Config{Mock: true}
	cfg.Weights = config.DefaultWeights()
	cfg.Mode = model.Mode(*modeFlag)

	switch cfg.Mode {
	case model.ModeThesis, model.ModeJournal:
	default:
		fmt.Fprintf(os.Stderr, "无效的 --mode 值 %q，可选 thesis / journal\n", cfg.Mode)
		os.Exit(1)
	}

	orch := flow.New(cfg, nil)

	step1JSON, err := orch.Run([]byte(`{}`))
	if err != nil {
		fmt.Fprintln(os.Stderr, "step1 失败:", err)
		os.Exit(1)
	}

	ctx := pipeline.NewContext(cfg.Mock, cfg.Strict)

	st2 := pipeline.Registry["step2_frame"]
	if st2 == nil {
		fmt.Fprintln(os.Stderr, "step2_frame 未注册")
		os.Exit(1)
	}
	step2JSON, err := st2.Run(ctx, step1JSON)
	if err != nil {
		fmt.Fprintln(os.Stderr, "step2 失败:", err)
		os.Exit(1)
	}

	st3 := pipeline.Registry["step3_litreview"]
	if st3 == nil {
		fmt.Fprintln(os.Stderr, "step3_litreview 未注册")
		os.Exit(1)
	}
	step3JSON, err := st3.Run(ctx, step2JSON)
	if err != nil {
		fmt.Fprintln(os.Stderr, "step3 失败:", err)
		os.Exit(1)
	}
	fmt.Println(string(step3JSON))
}
