package stata

import (
	"strings"
	"testing"
)

func TestValidate_MissingFields(t *testing.T) {
	var r StataRequest
	if err := r.Validate(); err == nil {
		t.Fatal("空请求应报错")
	}
}

func TestValidate_InvalidModel(t *testing.T) {
	r := StataRequest{DataPath: "x.dta", Dependent: "y", Core: "x", Model: "probit"}
	if err := r.Validate(); err == nil {
		t.Fatal("不支持的模型类型应报错")
	}
}

func TestGenerate_FE(t *testing.T) {
	req := StataRequest{
		DataPath:     "data.dta",
		OutputPath:   "out.csv",
		Dependent:    "Innov",
		Core:         "Digital",
		Controls:     []string{"Size", "Lev"},
		FixedEffects: []string{"firm", "year"},
		Cluster:      "firm",
		Model:        "fe",
		Label:        "基准回归",
	}
	out, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	for _, want := range []string{
		"data.dta",
		"out.csv",
		"Innov",
		"Digital",
		"Size",
		"Lev",
		"firm",
		"year",
		"基准回归",
		"export delimited",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("输出缺少 %q\n实际:\n%s", want, out)
		}
	}
}

func TestGenerate_NoFreeCommandRegion(t *testing.T) {
	// 安全回归：模板生成物中不应出现危险命令
	req := StataRequest{
		DataPath:   "data.dta",
		OutputPath: "out.csv",
		Dependent:  "y",
		Core:       "x",
		Model:      "fe",
	}
	out, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	for _, bad := range []string{"!cmd", "shell ", "!del", "rm ", "erase "} {
		if strings.Contains(out, bad) {
			t.Errorf("输出含危险命令 %q，违反安全红线\n实际:\n%s", bad, out)
		}
	}
}

func TestGenerate_MissingTemplate(t *testing.T) {
	req := StataRequest{
		DataPath:  "data.dta",
		Dependent: "y",
		Core:      "x",
		Model:     "iv", // 6.3 阶段只有 fe 模板
	}
	if _, err := Generate(req); err == nil {
		t.Fatal("缺模板应报错，而不是静默返回空")
	}
}
