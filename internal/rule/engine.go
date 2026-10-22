package rule

import (
	"strings"

	"paperagent/internal/model"
)

type Engine struct{}

func (e *Engine) ValidateTopic(t string, keywords []string) []model.Check {
	var out []model.Check
	for _, b := range []string{"初探", "浅析", "浅谈", "之我见"} {
		if strings.Contains(t, b) {
			out = append(out, model.Check{Level: "warn", Stage: "rule", Msg: "标题含降格词：" + b})
		}
	}
	if len(t) > 40 {
		out = append(out, model.Check{Level: "warn", Stage: "rule", Msg: "标题超过 40 字，建议收窄"})
	}
	if len(keywords) == 0 {
		out = append(out, model.Check{Level: "fail", Stage: "rule", Msg: "关键词为空"})
	}
	if len(out) == 0 {
		out = append(out, model.Check{Level: "pass", Stage: "rule", Msg: "论题合规性通过"})
	}
	return out
}

func (e *Engine) ValidateJournals(js []model.Journal, artType string) []model.Check {
	var out []model.Check
	if len(js) == 0 {
		out = append(out, model.Check{Level: "fail", Stage: "rule", Msg: "无匹配期刊，请放宽学科或关键词"})
		return out
	}
	if artType == "理论型" && js[0].EmpiricalOnly {
		out = append(out, model.Check{Level: "fail", Stage: "rule", Msg: "首推期刊栏目仅收实证，与所选文章类型冲突"})
	}
	out = append(out, model.Check{Level: "pass", Stage: "rule", Msg: "期刊规则校验通过"})
	return out
}
