package main

import (
	"flag"
	"fmt"
	"os"

	"paperagent/internal/config"
	"paperagent/internal/flow"
)

func main() {
	mock := flag.Bool("mock", false, "使用 mock 数据")
	flag.Parse()

	cfg := config.Config{}
	cfg.Mock = *mock
	cfg.Weights = config.DefaultWeights()

	orch := flow.New(cfg, nil)

	out, err := orch.Run([]byte(`{}`))
	if err != nil {
		fmt.Fprintln(os.Stderr, "step1 失败:", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
