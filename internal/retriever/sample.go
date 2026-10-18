package retriever

import "paperagent/internal/model"

func SamplePapers(mock bool) []model.SamplePaper {
	if !mock {
		// TODO: 真实检索
	}
	return []model.SamplePaper{
		{Title: "人工智能驱动的政府治理：研究述评与展望", Source: "中国行政管理", Year: 2024, Method: "文献计量+内容分析", URL: "https://example.com/1"},
		{Title: "政务大模型应用的风险识别与治理路径", Source: "公共管理学报", Year: 2025, Method: "多案例研究", URL: "https://example.com/2"},
	}
}