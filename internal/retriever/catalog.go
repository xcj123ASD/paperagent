package retriever

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"paperagent/internal/model"
)

func LoadCatalog(path string) ([]model.Journal, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comment = '#'
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("catalog empty")
	}
	var out []model.Journal
	for i, row := range rows[1:] {
		if len(row) < 11 {
			return nil, fmt.Errorf("row %d: too few columns", i+2)
		}
		j := model.Journal{
			Name:       strings.TrimSpace(row[0]),
			Abbr:       strings.TrimSpace(row[1]),
			Discipline: strings.TrimSpace(row[2]),
			Sub:        strings.TrimSpace(row[3]),
			WordMin:    atoi(row[7]),
			WordMax:    atoi(row[8]),
			TitleStyle: strings.TrimSpace(row[9]),
			Cycle:      strings.TrimSpace(row[10]),
		}
		for _, c := range strings.Split(row[5], "|") {
			if c = strings.TrimSpace(c); c != "" {
				j.Columns = append(j.Columns, c)
			}
		}
		switch strings.ToLower(strings.TrimSpace(row[6])) {
		case "true", "1", "yes":
			j.EmpiricalOnly = true
		}
		switch lv := strings.TrimSpace(row[4]); lv {
		case "CSSCI", "CSSCI扩展", "北大核心":
			j.Level = lv
		default:
			j.Level = "其他"
		}
		out = append(out, j)
	}
	return out, nil
}

func atoi(s string) int {
	var v int
	fmt.Sscanf(strings.TrimSpace(s), "%d", &v)
	return v
}

var MockCatalog = []model.Journal{
	{Name: "中国行政管理", Abbr: "ZGXZGL", Discipline: "公共管理", Sub: "数字治理", Level: "CSSCI",
		Columns: []string{"公共管理前沿", "数字治理"}, WordMin: 8000, WordMax: 12000,
		EmpiricalOnly: false, TitleStyle: "no_subtitle", Cycle: "月刊"},
	{Name: "电子政务", Abbr: "DZZW", Discipline: "公共管理", Sub: "电子政务", Level: "CSSCI",
		Columns: []string{"理论研究", "实践探索"}, WordMin: 7000, WordMax: 11000,
		EmpiricalOnly: false, TitleStyle: "no_subtitle", Cycle: "月刊"},
	{Name: "公共管理学报", Abbr: "GLGLXB", Discipline: "公共管理", Sub: "公共政策", Level: "CSSCI",
		Columns: []string{"论文"}, WordMin: 9000, WordMax: 14000,
		EmpiricalOnly: true, TitleStyle: "allow_subtitle", Cycle: "季刊"},
	{Name: "社会学研究", Abbr: "SHXBYJ", Discipline: "社会学", Sub: "社会分层", Level: "CSSCI",
		Columns: []string{"论文", "专题研讨"}, WordMin: 10000, WordMax: 15000,
		EmpiricalOnly: false, TitleStyle: "allow_subtitle", Cycle: "双月刊"},
	{Name: "管理世界", Abbr: "GLSJ", Discipline: "管理学", Sub: "组织与战略", Level: "CSSCI",
		Columns: []string{"实证研究", "综述"}, WordMin: 12000, WordMax: 18000,
		EmpiricalOnly: true, TitleStyle: "require_method", Cycle: "月刊"},
	{Name: "情报学报", Abbr: "QBXB", Discipline: "图书情报", Sub: "信息行为", Level: "CSSCI",
		Columns: []string{"理论探索", "技术应用"}, WordMin: 6000, WordMax: 10000,
		EmpiricalOnly: false, TitleStyle: "allow_subtitle", Cycle: "月刊"},
	{Name: "现代情报", Abbr: "XDQB", Discipline: "图书情报", Sub: "知识服务", Level: "北大核心",
		Columns: []string{"数字图书馆", "情报分析"}, WordMin: 5000, WordMax: 9000,
		EmpiricalOnly: false, TitleStyle: "allow_subtitle", Cycle: "月刊"},
}
