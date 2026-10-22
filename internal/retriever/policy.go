package retriever

import "paperagent/internal/model"

func PolicySources(mock bool) []model.PolicySource {
	if !mock {
		// TODO: 真实检索 gov.cn 白名单
	}
	return []model.PolicySource{
		{Name: "新一代人工智能发展规划", Org: "国务院", Date: "2017-07", URL: "https://www.gov.cn/zhengce/content/2017-07/20/content_5211996.htm"},
		{Name: "数字中国建设整体布局规划", Org: "中共中央、国务院", Date: "2023-02", URL: "https://www.gov.cn/zhengce/202302/content_6741030.htm"},
	}
}
