package verify

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

// ---- Check: papers 空 ----

func TestCheck_EmptyPapers_NonStrict(t *testing.T) {
	c := &Checker{Strict: false}
	checks := c.Check(nil, []model.PolicySource{{}})
	if !msgContains(checks, "样例文献为空") {
		t.Errorf("空 papers 应提示'样例文献为空'，实际: %+v", checks)
	}
	if hasLevel(checks, "fail") {
		t.Errorf("非 strict 模式空 papers 不应 fail，实际: %+v", checks)
	}
}

func TestCheck_EmptyPapers_Strict(t *testing.T) {
	c := &Checker{Strict: true}
	checks := c.Check(nil, []model.PolicySource{{}})
	if !msgContains(checks, "样例文献为空") {
		t.Errorf("空 papers 应提示，实际: %+v", checks)
	}
	// strict 时 lvl(true) = "fail"，所以空 papers 那条是 fail
	if !hasLevel(checks, "fail") {
		t.Errorf("strict 模式空 papers 应 fail，实际: %+v", checks)
	}
}

// ---- Check: papers 非空 ----

func TestCheck_PapersPresent(t *testing.T) {
	c := &Checker{Strict: false}
	checks := c.Check([]model.SamplePaper{{URL: "https://example.com"}}, []model.PolicySource{{}})
	if !msgContains(checks, "样例文献已就位") {
		t.Errorf("非空 papers 应提示'样例文献已就位'，实际: %+v", checks)
	}
}

// ---- Check: policies 空/非空 ----

func TestCheck_EmptyPolicies(t *testing.T) {
	c := &Checker{Strict: false}
	checks := c.Check([]model.SamplePaper{{URL: "https://x"}}, nil)
	if !msgContains(checks, "政策源为空") {
		t.Errorf("空 policies 应提示，实际: %+v", checks)
	}
	if !hasLevel(checks, "warn") {
		t.Errorf("空 policies 应 warn，实际: %+v", checks)
	}
}

func TestCheck_PoliciesPresent(t *testing.T) {
	c := &Checker{Strict: false}
	checks := c.Check([]model.SamplePaper{{URL: "https://x"}}, []model.PolicySource{{}})
	if !msgContains(checks, "政策源已就位") {
		t.Errorf("非空 policies 应提示，实际: %+v", checks)
	}
}

// ---- Check: strict 模式 URL 检查 ----

func TestCheck_StrictPaperWithoutURL(t *testing.T) {
	c := &Checker{Strict: true}
	checks := c.Check([]model.SamplePaper{{URL: ""}}, []model.PolicySource{{}})
	if !msgContains(checks, "strict 模式下文献必须带可访问链接") {
		t.Errorf("strict 模式文献无 URL 应提示，实际: %+v", checks)
	}
	if !hasLevel(checks, "fail") {
		t.Errorf("strict 模式文献无 URL 应 fail，实际: %+v", checks)
	}
}

func TestCheck_StrictPaperWithURL(t *testing.T) {
	c := &Checker{Strict: true}
	checks := c.Check([]model.SamplePaper{{URL: "https://example.com"}}, []model.PolicySource{{}})
	if msgContains(checks, "strict 模式下文献必须带可访问链接") {
		t.Errorf("有 URL 时不应提示 strict 检查，实际: %+v", checks)
	}
}

func TestCheck_NonStrictPaperWithoutURL(t *testing.T) {
	// 非 strict 不检查 URL
	c := &Checker{Strict: false}
	checks := c.Check([]model.SamplePaper{{URL: ""}}, []model.PolicySource{{}})
	if msgContains(checks, "strict 模式下文献必须带可访问链接") {
		t.Errorf("非 strict 模式不应检查 URL，实际: %+v", checks)
	}
}

// ---- Check: 所有 Check 的 Stage 都是 "verify" ----

func TestCheck_StageIsVerify(t *testing.T) {
	cases := []struct {
		name     string
		strict   bool
		papers   []model.SamplePaper
		policies []model.PolicySource
	}{
		{"empty_all_nonstrict", false, nil, nil},
		{"empty_all_strict", true, nil, nil},
		{"full", false, []model.SamplePaper{{URL: "x"}}, []model.PolicySource{{}}},
		{"strict_no_url", true, []model.SamplePaper{{URL: ""}}, []model.PolicySource{{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Checker{Strict: tc.strict}
			checks := c.Check(tc.papers, tc.policies)
			for _, ch := range checks {
				if ch.Stage != "verify" {
					t.Errorf("Check.Stage 应为 verify，实际 %q", ch.Stage)
				}
			}
		})
	}
}

// ---- lvl ----

func TestLvl(t *testing.T) {
	if got := lvl(true); got != "fail" {
		t.Errorf("lvl(true) = %q，期望 fail", got)
	}
	if got := lvl(false); got != "warn" {
		t.Errorf("lvl(false) = %q，期望 warn", got)
	}
}
