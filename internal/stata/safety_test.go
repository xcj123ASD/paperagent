package stata

import (
	"strings"
	"testing"
)

func TestCheckSafety_CleanTemplate(t *testing.T) {
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
	doText, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if err := CheckSafety(doText); err != nil {
		t.Errorf("干净模板被误拦: %v\n实际:\n%s", err, doText)
	}
}

func TestCheckSafety_RejectsDangerousCommands(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"bang_cmd", "!cmd /c dir"},
		{"bang_del", "!del important.dta"},
		{"bang_any", "!anything"},
		{"shell", "shell dir"},
		{"shell_upper", "SHELL DIR"},
		{"rm", "rm data.dta"},
		{"rm_tab", "rm\tdata.dta"},
		{"erase", "erase data.dta"},
		{"del", "del data.dta"},
		{"winexec", "winexec notepad.exe"},
		{"copy", "copy a.dta b.dta"},
		{"move", "move a.dta b.dta"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := CheckSafety(c.line)
			if err == nil {
				t.Errorf("危险命令未被拦截: %q", c.line)
			}
		})
	}
}

func TestCheckSafety_AllowsComments(t *testing.T) {
	// 注释里出现危险词不应误报
	text := "* 本脚本禁止使用 shell 和 !del 命令\n" +
		"* rm / erase 均不允许\n" +
		"reg y x\n"
	if err := CheckSafety(text); err != nil {
		t.Errorf("注释中的危险词被误报: %v", err)
	}
}

func TestCheckSafety_AllowsLegalVariables(t *testing.T) {
	// 合法变量名不应误报：rmse 不是 rm，delivery 不是 del
	text := "gen rmse = residual\n" +
		"gen delivery = 1\n" +
		"reg y rmse delivery\n"
	if err := CheckSafety(text); err != nil {
		t.Errorf("合法变量名被误报: %v", err)
	}
}

func TestCheckSafety_EmptyText(t *testing.T) {
	if err := CheckSafety(""); err != nil {
		t.Errorf("空文本不应报错: %v", err)
	}
}

func TestCheckSafety_OnlyWhitespace(t *testing.T) {
	if err := CheckSafety("  \n\t\n  \n"); err != nil {
		t.Errorf("纯空白不应报错: %v", err)
	}
}

func TestCheckSafety_ReportsLineNumber(t *testing.T) {
	text := "reg y x\n" +
		"!del important.dta\n"
	err := CheckSafety(text)
	if err == nil {
		t.Fatal("应报错")
	}
	if !strings.Contains(err.Error(), "第 2 行") {
		t.Errorf("错误信息应含行号'第 2 行'，实际: %v", err)
	}
}
