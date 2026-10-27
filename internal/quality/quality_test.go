package quality

import (
	"strings"
	"testing"

	"paperagent/internal/cite"
)

func TestCheckLitReview_骨架式_失败(t *testing.T) {
	md := "## 文献综述\n\n- 张三(2023)采用固定效应研究了《A》，发现 X。\n- 李四(2022)采用中介研究了《B》，发现 Y。\n- 王五(2021)采用DID研究了《C》，发现 Z。\n"
	ok, issues := CheckLitReview(md)
	if ok {
		t.Fatal("骨架式应判失败")
	}
	if len(issues) == 0 {
		t.Fatal("应有问题描述")
	}
}

func TestCheckLitReview_论述式_通过(t *testing.T) {
	md := "## 文献综述\n\n数字化转型如何影响创新，文献沿三条脉络展开。第一，技术能力视角，Goldfarb等(2019)指出……第二，成本视角……第三，资源视角……然而，现有研究仍有缺口。\n"
	ok, issues := CheckLitReview(md)
	if !ok {
		t.Fatalf("论述式应通过，问题：%v", issues)
	}
}

func TestCheckLitReview_占位符_失败(t *testing.T) {
	md := "## 文献综述\n\n第一段论述。\n\n（此处需补充该组文献的具体发现、方法与相互关系。）\n"
	ok, _ := CheckLitReview(md)
	if ok {
		t.Fatal("含骨架占位符应判失败")
	}
}

func TestCheckCitations_全部存在_通过(t *testing.T) {
	md := "正文提到张三(2023)和李四(2022)。\n## 参考文献\n[1] 张三. A. 2023.\n"
	refs := []cite.Reference{
		{Authors: []string{"张三"}, Year: 2023},
		{Authors: []string{"李四"}, Year: 2022},
	}
	ok, issues, warns := CheckCitations(md, refs)
	if !ok {
		t.Fatalf("应通过，问题：%v", issues)
	}
	if len(warns) != 0 {
		t.Fatalf("不应有 WARN，实际：%v", warns)
	}
}

func TestCheckCitations_缺失_失败(t *testing.T) {
	md := "正文提到王五(2020)。\n## 参考文献\n"
	refs := []cite.Reference{{Authors: []string{"张三"}, Year: 2023}}
	ok, issues, _ := CheckCitations(md, refs)
	if ok {
		t.Fatal("裸引用缺失应判失败")
	}
	if len(issues) == 0 || !strings.Contains(issues[0], "王五") {
		t.Fatalf("应报告王五缺失，实际：%v", issues)
	}
}

func TestCheckCitations_英文作者_通过(t *testing.T) {
	md := "Smith(2020)指出。\n## 参考文献\n"
	refs := []cite.Reference{{Authors: []string{"Smith J"}, Year: 2020}}
	ok, issues, _ := CheckCitations(md, refs)
	if !ok {
		t.Fatalf("英文作者应匹配，问题：%v", issues)
	}
}

func TestCheckCitations_有提示的幻觉_判WARN(t *testing.T) {
	md := "刘诗源等(2020)（此处需补充文献）关于税收激励的研究亦提示。\n## 参考文献\n"
	refs := []cite.Reference{{Authors: []string{"张三"}, Year: 2023}}
	ok, issues, warns := CheckCitations(md, refs)
	if !ok {
		t.Fatalf("有提示的幻觉应判通过，实际问题：%v", issues)
	}
	if len(warns) == 0 {
		t.Fatal("应产生 WARN")
	}
	if !strings.Contains(warns[0], "刘诗源") {
		t.Fatalf("WARN 应含刘诗源，实际：%v", warns)
	}
}

func TestCheckCitations_裸幻觉_判FAIL(t *testing.T) {
	md := "刘诗源等(2020)关于税收激励的研究提示。\n## 参考文献\n"
	refs := []cite.Reference{{Authors: []string{"张三"}, Year: 2023}}
	ok, issues, warns := CheckCitations(md, refs)
	if ok {
		t.Fatal("裸幻觉应判失败")
	}
	if len(warns) != 0 {
		t.Fatalf("不应有 WARN，实际：%v", warns)
	}
	if len(issues) == 0 || !strings.Contains(issues[0], "刘诗源") {
		t.Fatalf("应报告刘诗源裸幻觉，实际：%v", issues)
	}
}

func TestCheckPlaceholders_无残留_通过(t *testing.T) {
	md := "正文完整，无占位符。\n"
	n, _ := CheckPlaceholders(md)
	if n != 0 {
		t.Fatalf("应为 0，实际 %d", n)
	}
}

func TestCheckPlaceholders_有残留_计数(t *testing.T) {
	md := "第一段（此处需补充文献）。\n第二段（此处需填入数据）。\n"
	n, issues := CheckPlaceholders(md)
	if n != 2 {
		t.Fatalf("应为 2，实际 %d", n)
	}
	if len(issues) != 2 {
		t.Fatalf("应 2 条描述，实际 %d", len(issues))
	}
}

func TestAllPass_占位符不阻塞(t *testing.T) {
	r := Report{LitReviewOK: true, CitationOK: true, PlaceholderCount: 5}
	if !r.AllPass() {
		t.Fatal("占位符不应阻塞 AllPass")
	}
}

func TestCheckCitations_拼写变形_判WARN(t *testing.T) {
	md := "Brynjansson(2000)指出。\n## 参考文献\n"
	refs := []cite.Reference{{Authors: []string{"Brynjolfsson E"}, Year: 2000}}
	ok, issues, warns := CheckCitations(md, refs)
	if !ok {
		t.Fatalf("拼写变形应判 WARN 不判 FAIL，实际问题：%v", issues)
	}
	if len(warns) == 0 {
		t.Fatal("应产生 WARN")
	}
}

func TestCheckCitations_中文单姓_判WARN(t *testing.T) {
	md := "刘等(2023)发现。\n## 参考文献\n"
	refs := []cite.Reference{{Authors: []string{"Liu M"}, Year: 2023}}
	ok, issues, warns := CheckCitations(md, refs)
	if !ok {
		t.Fatalf("中文单姓缩写应判 WARN，实际问题：%v", issues)
	}
	if len(warns) == 0 {
		t.Fatal("应产生 WARN")
	}
}

func TestCheckCitations_英文全名缩写_通过(t *testing.T) {
	md := "Teece D J(1997)指出。\n## 参考文献\n"
	refs := []cite.Reference{{Authors: []string{"Teece D J"}, Year: 1997}}
	ok, issues, _ := CheckCitations(md, refs)
	if !ok {
		t.Fatalf("英文全名缩写应匹配，问题：%v", issues)
	}
}

func TestCheckCitations_多作者全名缩写_通过(t *testing.T) {
	md := "Teece D J(1997)和Cohen W M(1990)均指出。\n## 参考文献\n"
	refs := []cite.Reference{
		{Authors: []string{"Teece D J"}, Year: 1997},
		{Authors: []string{"Cohen W M"}, Year: 1990},
	}
	ok, issues, _ := CheckCitations(md, refs)
	if !ok {
		t.Fatalf("多作者全名缩写应全匹配，问题：%v", issues)
	}
}
