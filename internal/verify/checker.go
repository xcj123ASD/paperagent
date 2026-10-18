package verify

import "paperagent/internal/model"

type Checker struct{ Strict bool }

func (c *Checker) Check(papers []model.SamplePaper, policies []model.PolicySource) []model.Check {
	var out []model.Check
	if len(papers) == 0 {
		out = append(out, model.Check{Level: lvl(c.Strict), Stage: "verify", Msg: "样例文献为空，后续引用无法核验"})
	} else {
		out = append(out, model.Check{Level: "pass", Stage: "verify", Msg: "样例文献已就位"})
	}
	if len(policies) == 0 {
		out = append(out, model.Check{Level: "warn", Stage: "verify", Msg: "政策源为空，政策类主张需人工补证"})
	} else {
		out = append(out, model.Check{Level: "pass", Stage: "verify", Msg: "政策源已就位"})
	}
	if c.Strict && len(papers) > 0 && papers[0].URL == "" {
		out = append(out, model.Check{Level: "fail", Stage: "verify", Msg: "strict 模式下文献必须带可访问链接"})
	}
	return out
}

func lvl(s bool) string {
	if s {
		return "fail"
	}
	return "warn"
}