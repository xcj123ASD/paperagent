package cite

import (
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"net/url"
"strings"
"time"
)

// OpenAlexVerifier 通过 OpenAlex 公开 API 做单向校验。
// 不需要 API Key，但需要在 User-Agent 里带上联系方式（OpenAlex 礼貌池要求）。
type OpenAlexVerifier struct {
BaseURL    string
HTTPClient *http.Client
Mailto     string // 可选，填邮箱可进礼貌池，请求更快
}

func NewOpenAlexVerifier(mailto string) *OpenAlexVerifier {
return &OpenAlexVerifier{
BaseURL: "https://api.openalex.org",
HTTPClient: &http.Client{
Timeout: 15 * time.Second,
},
Mailto: mailto,
}
}

func (v *OpenAlexVerifier) Name() string { return "OpenAlex" }

func (v *OpenAlexVerifier) Verify(ctx context.Context, ref Reference) (VerifyResult, error) {
// 优先用 DOI 精确查询
if ref.DOI != "" {
return v.verifyByDOI(ctx, ref)
}
// DOI 缺失，退化到标题搜索
if ref.Title != "" {
return v.verifyByTitle(ctx, ref)
}
return VerifyResult{
Status:  VerifyNotFound,
Message: "DOI 和标题均为空，无法校验",
}, nil
}

func (v *OpenAlexVerifier) verifyByDOI(ctx context.Context, ref Reference) (VerifyResult, error) {
doi := strings.TrimPrefix(ref.DOI, "https://doi.org/")
doi = strings.TrimPrefix(doi, "http://doi.org/")
endpoint := fmt.Sprintf("%s/works/doi:%s", v.BaseURL, url.PathEscape(doi))

body, status, err := v.get(ctx, endpoint)
if err != nil {
return VerifyResult{Status: VerifyError, Message: err.Error()}, nil
}
if status == http.StatusNotFound {
return VerifyResult{
Status:  VerifyNotFound,
Message: fmt.Sprintf("DOI %s 在 OpenAlex 中不存在", doi),
}, nil
}
if status != http.StatusOK {
return VerifyResult{
Status:  VerifyError,
Message: fmt.Sprintf("OpenAlex 返回 %d", status),
}, nil
}

var work openAlexWork
if err := json.Unmarshal(body, &work); err != nil {
return VerifyResult{Status: VerifyError, Message: "解析 OpenAlex 响应失败: " + err.Error()}, nil
}

matched := work.toReference()
return VerifyResult{
Status:     VerifyFound,
Matched:    &matched,
Confidence: 1.0,
Message:    "DOI 精确匹配",
}, nil
}

func (v *OpenAlexVerifier) verifyByTitle(ctx context.Context, ref Reference) (VerifyResult, error) {
q := url.Values{}
q.Set("filter", "title.search:"+ref.Title)
q.Set("per-page", "5")
endpoint := fmt.Sprintf("%s/works?%s", v.BaseURL, q.Encode())

body, status, err := v.get(ctx, endpoint)
if err != nil {
return VerifyResult{Status: VerifyError, Message: err.Error()}, nil
}
if status != http.StatusOK {
return VerifyResult{Status: VerifyError, Message: fmt.Sprintf("OpenAlex 返回 %d", status)}, nil
}

var resp struct {
Results []openAlexWork `json:"results"`
}
if err := json.Unmarshal(body, &resp); err != nil {
return VerifyResult{Status: VerifyError, Message: "解析 OpenAlex 响应失败: " + err.Error()}, nil
}
if len(resp.Results) == 0 {
return VerifyResult{Status: VerifyNotFound, Message: "标题无匹配"}, nil
}

top := resp.Results[0]
matched := top.toReference()
return VerifyResult{
Status:     VerifyFound,
Matched:    &matched,
Confidence: 0.6,
Message:    "标题匹配（非精确，建议人工核对）",
}, nil
}

func (v *OpenAlexVerifier) get(ctx context.Context, endpoint string) ([]byte, int, error) {
req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
if err != nil {
return nil, 0, err
}
ua := "paperagent/0.1"
if v.Mailto != "" {
ua += " (mailto:" + v.Mailto + ")"
}
req.Header.Set("User-Agent", ua)
req.Header.Set("Accept", "application/json")

resp, err := v.HTTPClient.Do(req)
if err != nil {
return nil, 0, err
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, resp.StatusCode, err
}
return body, resp.StatusCode, nil
}

// openAlexWork OpenAlex /works 返回结构（只取需要的字段）。
type openAlexWork struct {
DOI             string `json:"doi"`
Title           string `json:"title"`
PublicationYear int    `json:"publication_year"`
Type            string `json:"type"`
Authorships     []struct {
Author struct {
DisplayName string `json:"display_name"`
} `json:"author"`
} `json:"authorships"`
PrimaryLocation struct {
Source struct {
DisplayName string `json:"display_name"`
} `json:"source"`
} `json:"primary_location"`
Biblio struct {
Volume    string `json:"volume"`
Issue     string `json:"issue"`
FirstPage string `json:"first_page"`
LastPage  string `json:"last_page"`
} `json:"biblio"`
}

func (w openAlexWork) toReference() Reference {
r := Reference{
Title:   w.Title,
Year:    w.PublicationYear,
DOI:     strings.TrimPrefix(w.DOI, "https://doi.org/"),
Volume:  w.Biblio.Volume,
Issue:   w.Biblio.Issue,
Lang:    detectLang(w.Title),
Type:    mapOpenAlexType(w.Type),
Container: w.PrimaryLocation.Source.DisplayName,
}
for _, a := range w.Authorships {
if a.Author.DisplayName != "" {
r.Authors = append(r.Authors, a.Author.DisplayName)
}
}
if w.Biblio.FirstPage != "" {
if w.Biblio.LastPage != "" && w.Biblio.LastPage != w.Biblio.FirstPage {
r.Pages = w.Biblio.FirstPage + "-" + w.Biblio.LastPage
} else {
r.Pages = w.Biblio.FirstPage
}
}
return r
}

func mapOpenAlexType(t string) string {
switch t {
case "article", "preprint":
return "article-journal"
case "book":
return "book"
case "book-chapter":
return "chapter"
case "dissertation":
return "thesis"
case "report":
return "report"
default:
return "article-journal"
}
}

// detectLang 简单判断：标题含 CJK 字符视为 zh，否则 en。
func detectLang(s string) string {
for _, r := range s {
if r >= 0x4E00 && r <= 0x9FFF {
return "zh"
}
}
return "en"
}