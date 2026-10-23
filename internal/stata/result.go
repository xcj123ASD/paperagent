package stata

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Result 是一行回归结果，对应模板 export 的 CSV 单行：
// variable,coef,se,pvalue
type Result struct {
	Variable string  // 变量名，如 "Digital"
	Coef     float64 // 系数
	SE       float64 // 标准误
	PValue   float64 // 双侧 p 值
}

// ParseCSV 读取模板生成的 CSV，返回解析后的 Result。
//
// 只解析 CSV，不解析 .log——log 格式随 Stata 版本/设置变化，CSV 稳定。
// 任何异常（文件不存在、缺列、数值非法、Stata 缺失值 "."）都返回明确 error，
// 不静默返回零值，避免把"回归没收敛"误当成"系数为 0"。
func ParseCSV(path string) (Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return Result{}, fmt.Errorf("stata: 打开结果 CSV 失败: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return Result{}, fmt.Errorf("stata: 解析 CSV 失败: %w", err)
	}
	if len(records) < 2 {
		return Result{}, fmt.Errorf("stata: 结果 CSV 至少需要表头+1 行数据，实际 %d 行", len(records))
	}

	header := records[0]
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.TrimSpace(strings.ToLower(h))] = i
	}
	for _, col := range []string{"variable", "coef", "se", "pvalue"} {
		if _, ok := idx[col]; !ok {
			return Result{}, fmt.Errorf("stata: 结果 CSV 缺少列 %q，实际表头: %v", col, header)
		}
	}

	row := records[1]
	if len(row) < len(header) {
		return Result{}, fmt.Errorf("stata: 数据行列数 %d 少于表头 %d", len(row), len(header))
	}

	parseFloat := func(col string) (float64, error) {
		raw := strings.TrimSpace(row[idx[col]])
		if raw == "" || raw == "." {
			return 0, fmt.Errorf("列 %q 为 Stata 缺失值 %q，回归可能未收敛", col, raw)
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, fmt.Errorf("列 %q 数值 %q 无法解析: %w", col, raw, err)
		}
		return v, nil
	}

	coef, err := parseFloat("coef")
	if err != nil {
		return Result{}, err
	}
	se, err := parseFloat("se")
	if err != nil {
		return Result{}, err
	}
	pvalue, err := parseFloat("pvalue")
	if err != nil {
		return Result{}, err
	}

	return Result{
		Variable: strings.TrimSpace(row[idx["variable"]]),
		Coef:     coef,
		SE:       se,
		PValue:   pvalue,
	}, nil
}
