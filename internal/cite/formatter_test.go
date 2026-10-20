package cite

import (
"strings"
"testing"
)

func TestGBT7714_Name(t *testing.T) {
f := GBT7714{}
if f.Name() != "GB/T 7714-2015" {
t.Errorf("Name 应为 GB/T 7714-2015，实际 %s", f.Name())
}
}

func TestGBT7714_Empty(t *testing.T) {
f := GBT7714{}
out, err := f.Format(nil)
if err != nil {
t.Fatalf("不应报错: %v", err)
}
if out != "" {
t.Errorf("空输入应返回空字符串，实际 %q", out)
}
}

func TestGBT7714_Journal(t *testing.T) {
f := GBT7714{}
refs := []Reference{{
Type:      "article-journal",
Authors:   []string{"张三", "李四"},
Title:     "数字政府建设的路径研究",
Container: "中国行政管理",
Year:      2023,
Volume:    "12",
Issue:     "3",
Pages:     "45-58",
Lang:      "zh",
}}
out, err := f.Format(refs)
if err != nil {
t.Fatalf("不应报错: %v", err)
}
want := "[1] 张三, 李四. 数字政府建设的路径研究[J]. 中国行政管理, 2023, 12(3): 45-58."
if out != want {
t.Errorf("输出不符\n期望: %s\n实际: %s", want, out)
}
}

func TestGBT7714_JournalEnglish(t *testing.T) {
f := GBT7714{}
refs := []Reference{{
Type:      "article-journal",
Authors:   []string{"John Smith", "Mary Jane Watson"},
Title:     "Digital Governance",
Container: "Public Administration Review",
Year:      2022,
Volume:    "82",
Issue:     "1",
Pages:     "10-25",
Lang:      "en",
}}
out, _ := f.Format(refs)
want := "[1] Smith J, Watson M J. Digital Governance[J]. Public Administration Review, 2022, 82(1): 10-25."
if out != want {
t.Errorf("输出不符\n期望: %s\n实际: %s", want, out)
}
}

func TestGBT7714_Book(t *testing.T) {
f := GBT7714{}
refs := []Reference{{
Type:      "book",
Authors:   []string{"王五"},
Title:     "公共管理导论",
Publisher: "北京大学出版社",
Year:      2021,
Lang:      "zh",
}}
out, _ := f.Format(refs)
want := "[1] 王五. 公共管理导论[M]. 北京大学出版社, 2021."
if out != want {
t.Errorf("输出不符\n期望: %s\n实际: %s", want, out)
}
}

func TestGBT7714_Thesis(t *testing.T) {
f := GBT7714{}
refs := []Reference{{
Type:      "thesis",
Authors:   []string{"赵六"},
Title:     "数字治理的机制研究",
Publisher: "清华大学",
Year:      2020,
Lang:      "zh",
}}
out, _ := f.Format(refs)
want := "[1] 赵六. 数字治理的机制研究[D]. 清华大学, 2020."
if out != want {
t.Errorf("输出不符\n期望: %s\n实际: %s", want, out)
}
}

func TestGBT7714_Web(t *testing.T) {
f := GBT7714{}
refs := []Reference{{
Type:       "web",
Authors:    []string{"国务院"},
Title:      "关于加强数字政府建设的指导意见",
Year:       2022,
URL:        "http://www.gov.cn/xxx",
AccessDate: "2023-01-15",
Lang:       "zh",
}}
out, _ := f.Format(refs)
want := "[1] 国务院. 关于加强数字政府建设的指导意见[EB/OL]. (2022)[2023-01-15]. http://www.gov.cn/xxx."
if out != want {
t.Errorf("输出不符\n期望: %s\n实际: %s", want, out)
}
}

func TestGBT7714_ManyAuthors(t *testing.T) {
f := GBT7714{}
refs := []Reference{{
Type:      "article-journal",
Authors:   []string{"作者一", "作者二", "作者三", "作者四"},
Title:     "测试标题",
Container: "测试期刊",
Year:      2023,
Lang:      "zh",
}}
out, _ := f.Format(refs)
if !strings.Contains(out, "作者一, 作者二, 作者三, 等") {
t.Errorf("4 个作者应截断为前 3 + 等，实际: %s", out)
}
}

func TestGBT7714_MultipleRefs(t *testing.T) {
f := GBT7714{}
refs := []Reference{
{Type: "book", Authors: []string{"甲"}, Title: "书一", Publisher: "出版社A", Year: 2020, Lang: "zh"},
{Type: "book", Authors: []string{"乙"}, Title: "书二", Publisher: "出版社B", Year: 2021, Lang: "zh"},
}
out, _ := f.Format(refs)
if !strings.Contains(out, "[1]") || !strings.Contains(out, "[2]") {
t.Errorf("多条应编号，实际: %s", out)
}
}

func TestGBT7714_UnknownType(t *testing.T) {
f := GBT7714{}
refs := []Reference{{Type: "unknown", Authors: []string{"甲"}, Title: "测试", Container: "刊", Year: 2020, Lang: "zh"}}
out, _ := f.Format(refs)
if !strings.Contains(out, "[J]") {
t.Errorf("未知类型应默认按期刊格式，实际: %s", out)
}
}

func TestGBT7714_NoAuthor(t *testing.T) {
f := GBT7714{}
refs := []Reference{{Type: "book", Title: "佚名之书", Publisher: "某社", Year: 2020, Lang: "zh"}}
out, _ := f.Format(refs)
if !strings.Contains(out, "佚名") {
t.Errorf("无作者应显示佚名，实际: %s", out)
}
}