package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"paperagent/internal/stata"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("stata", flag.ContinueOnError)

	requestFlag := fs.String("request", "", "StataRequest JSON 文件路径（给定时忽略其他输入参数）")
	dataFlag := fs.String("data", "", ".dta 数据文件路径")
	dependentFlag := fs.String("dependent", "", "被解释变量")
	coreFlag := fs.String("core", "", "核心解释变量")
	controlsFlag := fs.String("controls", "", "控制变量，逗号分隔")
	fixedEffectsFlag := fs.String("fixed-effects", "", "固定效应变量，逗号分隔")
	clusterFlag := fs.String("cluster", "", "聚类变量")
	modelFlag := fs.String("model", "fe", "模型类型：ols / fe / did / iv")
	instrumentFlag := fs.String("instrument", "", "工具变量（model=iv 时用）")
	didTreatmentFlag := fs.String("did-treatment", "", "DID 处理组变量")
	didTimeFlag := fs.String("did-time", "", "DID 政策时间变量")
	labelFlag := fs.String("label", "", "任务标签（进注释）")
	outputDoFlag := fs.String("output-do", "regression.do", ".do 输出路径")
	outputCSVFlag := fs.String("output-csv", "result.csv", "结果 CSV 输出路径")
	runStataFlag := fs.Bool("run-stata", false, "是否真执行 Stata（默认只生成 .do）")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	var req stata.StataRequest
	if *requestFlag != "" {
		data, err := os.ReadFile(*requestFlag)
		if err != nil {
			return fmt.Errorf("读取 request 文件失败: %w", err)
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return fmt.Errorf("解析 request JSON 失败: %w", err)
		}
	} else {
		req = stata.StataRequest{
			DataPath:     *dataFlag,
			OutputPath:   *outputCSVFlag,
			Dependent:    *dependentFlag,
			Core:         *coreFlag,
			Controls:     splitCSV(*controlsFlag),
			FixedEffects: splitCSV(*fixedEffectsFlag),
			Cluster:      *clusterFlag,
			Model:        *modelFlag,
			Instrument:   *instrumentFlag,
			DIDTreatment: *didTreatmentFlag,
			DIDTime:      *didTimeFlag,
			Label:        *labelFlag,
		}
	}

	if req.OutputPath == "" {
		req.OutputPath = *outputCSVFlag
	}

	doText, err := stata.Generate(req)
	if err != nil {
		return fmt.Errorf("生成 .do 失败: %w", err)
	}

	if err := stata.CheckSafety(doText); err != nil {
		return fmt.Errorf("安全检查未通过: %w", err)
	}

	if err := os.WriteFile(*outputDoFlag, []byte(doText), 0644); err != nil {
		return fmt.Errorf("写 .do 失败: %w", err)
	}
	fmt.Printf(".do written: %s\n", *outputDoFlag)

	if !*runStataFlag {
		fmt.Println("未执行 Stata（默认只生成 .do）。加 --run-stata 才会真执行。")
		return nil
	}

	runner, err := stata.NewRunner()
	if err != nil {
		return fmt.Errorf("初始化 Stata runner 失败: %w", err)
	}

	out, err := runner.Run(context.Background(), *outputDoFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Stata 执行输出:\n%s\n", out)
		return fmt.Errorf("Stata 执行失败: %w", err)
	}

	result, err := stata.ParseCSV(req.OutputPath)
	if err != nil {
		return fmt.Errorf("解析结果 CSV 失败: %w", err)
	}

	fmt.Printf("Result: %s coef=%.6f se=%.6f pvalue=%.4f\n",
		result.Variable, result.Coef, result.SE, result.PValue)
	return nil
}

// splitCSV 把逗号分隔字符串拆成切片，trim 并去空项。
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
