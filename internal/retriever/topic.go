package retriever

import "paperagent/internal/model"

func GenerateTopics(disc, sub string, mock bool) []model.TopicCandidate {
	if !mock {
		// TODO: 调 LLM 生成
	}
	return []model.TopicCandidate{
		{Title: "生成式人工智能赋能政务服务的风险治理机制研究", Disc: "公共管理", Keywords: []string{"生成式AI", "政务服务", "风险治理"}, Type_: "实证型", Score: 0.91, Why: "政策热度高且有可获取的案例数据"},
		{Title: "数据要素市场化配置的政策演进与地方实践", Disc: "公共管理", Keywords: []string{"数据要素", "政策演进", "地方实践"}, Type_: "政策研究型", Score: 0.84, Why: "政策文本充足，适合做内容分析"},
		{Title: "算法问责的制度逻辑与规范建构", Disc: "法学", Keywords: []string{"算法问责", "制度逻辑", "规范建构"}, Type_: "理论型", Score: 0.78, Why: "理论空间大但实证数据难获取"},
	}
}
