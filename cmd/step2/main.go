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

	st := pipeline.Registry["step2_frame"]
	if st == nil {
		fmt.Fprintln(os.Stderr, "step2_frame 未注册")
		os.Exit(1)
	}
	ctx := pipeline.NewContext(cfg.Mock, cfg.Strict)
	step2JSON, err := st.Run(ctx, step1JSON)
	if err != nil {
		fmt.Fprintln(os.Stderr, "step2 失败:", err)
		os.Exit(1)
	}
	fmt.Println(string(step2JSON))
}