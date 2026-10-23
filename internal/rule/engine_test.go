package rule

import (
	"strings"
	"testing"

	"paperagent/internal/model"
)

// hasLevel 判断 checks 中是否有指定 Level 的项。
func hasLevel(checks []model.Check, level string) bool {
	for _, c := range checks {
		if c.Level == level {
			return true
		}
	}
	return false
}

// msgContains 判断 checks 中是否有 Msg 含指定子串的项。
func msgContains(checks []model.Check, sub string) bool {
	for _, c := range checks {
		if strings.Contains(c.Msg, sub) {
			return true
		}
	}
	return false
}

// ---- ValidateTopic ----

func TestValidateTopic_Pass(t *testing.T) {
	checks := new(Engine).ValidateTopic("数字化转型对制造业创新绩效的影响", []string{"数字化", "创新"})
	if !hasLevel(checks, "pass") {
		t.Errorf("正常选题应出 pass，实际: %+v", checks)
	}
	if hasLevel(checks, "fail") || hasLevel(checks, "warn") {
		t.Errorf("正常选题不应有 fail/warn，实际: %+v", checks)
	}
}

func TestValidateTopic_EmptyKeywords(t *testing.T) {
	checks := new(Engine).ValidateTopic("正常标题", nil)
	if !hasLevel(checks, "fail") {
		t.Errorf("空关键词应出 fail，实际: %+v", checks)
	}
	if !msgContains(checks, "关键词为空") {
		t.Errorf("错误信息应含'关键词为空'，实际: %+v", checks)
	}
}

func TestValidateTopic_BannedWords(t *testing.T) {
	cases := []string{"初探", "浅析", "浅谈", "之我见"}
	for _, w := range cases {
		t.Run(w, func(t *testing.T) {
			checks := new(Engine).ValidateTopic("XX 问题"+w, []string{"k"})
			if !hasLevel(checks, "warn") {
				t.Errorf("含禁用词 %q 应出 warn，实际: %+v", w, checks)
			}
			if !msgContains(checks, w) {
				t.Errorf("错误信息应含 %q，实际: %+v", w, checks)
			}
		})
	}
}

func TestValidateTopic_TooLong(t *testing.T) {
	// 41 个中文字符，超过 40
	long := strings.Repeat("测", 41)
	checks := new(Engine).ValidateTopic(long, []string{"k"})
	if !msgContains(checks, "超过 40 字") {
		t.Errorf("超长标题应出警告，实际: %+v", checks)
	}
}

func TestValidateTopic_Exactly40(t *testing.T) {
	// 边界：40 字不应触发警告
	exact := strings.Repeat("测", 40)
	checks := new(Engine).ValidateTopic(exact, []string{"k"})
	if msgContains(checks, "超过 40 字") {
		t.Errorf("正好 40 字不应触发超长警告，实际: %+v", checks)
	}
}

// ---- ValidateJournals ----

func TestValidateJournals_Empty(t *testing.T) {
	checks := new(Engine).ValidateJournals(nil, "")
	if !hasLevel(checks, "fail") {
		t.Errorf("空期刊列表应出 fail，实际: %+v", checks)
	}
	if !msgContains(checks, "无匹配期刊") {
		t.Errorf("错误信息应含'无匹配期刊'，实际: %+v", checks)
	}
}

func TestValidateJournals_EmptySlice(t *testing.T) {
	checks := new(Engine).ValidateJournals([]model.Journal{}, "")
	if !hasLevel(checks, "fail") {
		t.Errorf("空切片也应出 fail，实际: %+v", checks)
	}
}

func TestValidateJournals_Pass(t *testing.T) {
	js := []model.Journal{{Name: "测试期刊"}}
	checks := new(Engine).ValidateJournals(js, "实证型")
	if !hasLevel(checks, "pass") {
		t.Errorf("正常期刊应出 pass，实际: %+v", checks)
	}
}

func TestValidateJournals_TheoreticalVsEmpiricalOnly(t *testing.T) {
	// artType=理论型 + 首推期刊仅收实证 → fail
	js := []model.Journal{{Name: "纯实证期刊", EmpiricalOnly: true}}
	checks := new(Engine).ValidateJournals(js, "理论型")
	if !hasLevel(checks, "fail") {
		t.Errorf("理论型选实证期刊应出 fail，实际: %+v", checks)
	}
	if !msgContains(checks, "仅收实证") {
		t.Errorf("错误信息应含'仅收实证'，实际: %+v", checks)
	}
}

func TestValidateJournals_EmpiricalVsEmpiricalOnly(t *testing.T) {
	// artType=实证型 + 仅收实证期刊 → 不冲突，应无 fail
	js := []model.Journal{{Name: "纯实证期刊", EmpiricalOnly: true}}
	checks := new(Engine).ValidateJournals(js, "实证型")
	if hasLevel(checks, "fail") {
		t.Errorf("实证型选实证期刊不应 fail，实际: %+v", checks)
	}
	if !hasLevel(checks, "pass") {
		t.Errorf("应出 pass，实际: %+v", checks)
	}
}

func TestValidateJournals_CheckFields(t *testing.T) {
	// 所有 Check 的 Stage 都应为 "rule"
	checks := new(Engine).ValidateJournals([]model.Journal{{Name: "x"}}, "")
	for _, c := range checks {
		if c.Stage != "rule" {
			t.Errorf("Check.Stage 应为 rule，实际 %q", c.Stage)
		}
	}
	checks = new(Engine).ValidateTopic("x", []string{"k"})
	for _, c := range checks {
		if c.Stage != "rule" {
			t.Errorf("ValidateTopic 的 Check.Stage 应为 rule，实际 %q", c.Stage)
		}
	}
}
