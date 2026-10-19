package main

import (
	"fmt"
	"os"

	"paperagent/internal/config"
	"paperagent/internal/flow"
	"paperagent/internal/pipeline"
)

func main() {
	cfg := config.Config{Mock: true}
	cfg.Weights = config.DefaultWeights()

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