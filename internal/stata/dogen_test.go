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
	// 安全回归：模板生成物必须过 CheckSafety。
	// 危险命令列表由 safety.go 的 CheckSafety 统一维护，此处不重复定义。
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
	if err := CheckSafety(out); err != nil {
		t.Errorf("模板生成物含危险命令，违反安全红线: %v\n实际:\n%s", err, out)
	}
}

func TestGenerate_IVMissingInstrument(t *testing.T) {
	// 原名 TestGenerate_MissingTemplate 名不副实：
	// 四个模型（ols/fe/did/iv）都有模板，"缺模板"是死路径。
	// 实际测的是 model=iv 缺 instrument 被 Validate 拦截。
	req := StataRequest{
		DataPath:  "data.dta",
		Dependent: "y",
		Core:      "x",
		Model:     "iv", // iv 需要 instrument，缺失时 Validate 应拦截
	}
	_, err := Generate(req)
	if err == nil {
		t.Fatal("model=iv 缺 instrument 应报错")
	}
	if !strings.Contains(err.Error(), "instrument 必填") {
		t.Errorf("错误信息应含'instrument 必填'，实际: %v", err)
	}
}
