package cite

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAlex_VerifyByDOI_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"doi": "https://doi.org/10.1038/nature12373",
			"title": "Nanometre-scale thermometry in a living cell",
			"publication_year": 2013,
			"type": "article",
			"authorships": [
				{"author": {"display_name": "G Kucsko"}},
				{"author": {"display_name": "P C Maurer"}}
			],
			"primary_location": {"source": {"display_name": "Nature"}},
			"biblio": {"volume": "500", "issue": "7460", "first_page": "54", "last_page": "58"}
		}`))
	}))
	defer srv.Close()

	v := NewOpenAlexVerifier("")
	v.BaseURL = srv.URL

	ref := Reference{DOI: "10.1038/nature12373"}
	res, err := v.Verify(context.Background(), ref)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if res.Status != VerifyFound {
		t.Fatalf("应为 VerifyFound，实际 %v，消息: %s", res.Status, res.Message)
	}
	if res.Confidence != 1.0 {
		t.Errorf("DOI 精确匹配 confidence 应为 1.0，实际 %v", res.Confidence)
	}
	if res.Matched == nil {
		t.Fatal("Matched 不应为 nil")
	}
	if res.Matched.Title != "Nanometre-scale thermometry in a living cell" {
		t.Errorf("标题不符: %s", res.Matched.Title)
	}
	if res.Matched.Year != 2013 {
		t.Errorf("年份不符: %d", res.Matched.Year)
	}
	if res.Matched.Pages != "54-58" {
		t.Errorf("页码不符: %s", res.Matched.Pages)
	}
	if res.Matched.DOI != "10.1038/nature12373" {
		t.Errorf("DOI 应去掉前缀，实际: %s", res.Matched.DOI)
	}
	if res.Matched.Lang != "en" {
		t.Errorf("英文标题应识别为 en，实际: %s", res.Matched.Lang)
	}
}

func TestOpenAlex_VerifyByDOI_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	v := NewOpenAlexVerifier("")
	v.BaseURL = srv.URL

	ref := Reference{DOI: "10.9999/nonexistent"}
	res, err := v.Verify(context.Background(), ref)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if res.Status != VerifyNotFound {
		t.Errorf("应为 VerifyNotFound，实际 %v", res.Status)
	}
}

func TestOpenAlex_VerifyByTitle_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"results": [{
				"doi": "https://doi.org/10.1234/test",
				"title": "Digital Governance in China",
				"publication_year": 2023,
				"type": "article",
				"authorships": [{"author": {"display_name": "John Smith"}}],
				"primary_location": {"source": {"display_name": "Public Administration Review"}},
				"biblio": {"volume": "5", "issue": "2", "first_page": "10", "last_page": "20"}
			}]
		}`))
	}))
	defer srv.Close()

	v := NewOpenAlexVerifier("")
	v.BaseURL = srv.URL

	ref := Reference{Title: "Digital Governance in China"}
	res, err := v.Verify(context.Background(), ref)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if res.Status != VerifyFound {
		t.Fatalf("应为 VerifyFound，实际 %v", res.Status)
	}
	if res.Confidence >= 1.0 {
		t.Errorf("标题匹配 confidence 应 < 1.0，实际 %v", res.Confidence)
	}
	if res.Matched.Lang != "en" {
		t.Errorf("英文标题应识别为 en，实际: %s", res.Matched.Lang)
	}
}

func TestOpenAlex_VerifyByTitle_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": []}`))
	}))
	defer srv.Close()

	v := NewOpenAlexVerifier("")
	v.BaseURL = srv.URL

	ref := Reference{Title: "Nonexistent Title xyzabc"}
	res, err := v.Verify(context.Background(), ref)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if res.Status != VerifyNotFound {
		t.Errorf("应为 VerifyNotFound，实际 %v", res.Status)
	}
}

func TestOpenAlex_ChineseNoDOISkipped(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	v := NewOpenAlexVerifier("")
	v.BaseURL = srv.URL

	ref := Reference{Title: "数字治理的中国路径"}
	res, err := v.Verify(context.Background(), ref)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if res.Status != VerifyNotFound {
		t.Errorf("中文无 DOI 应返回 VerifyNotFound，实际 %v", res.Status)
	}
	if called {
		t.Errorf("中文无 DOI 不应发起网络请求，实际发出了")
	}
}

func TestOpenAlex_NoDOINoTitle(t *testing.T) {
	v := NewOpenAlexVerifier("")
	ref := Reference{}
	res, err := v.Verify(context.Background(), ref)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if res.Status != VerifyNotFound {
		t.Errorf("无 DOI 无标题应返回 VerifyNotFound，实际 %v", res.Status)
	}
}

func TestOpenAlex_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	v := NewOpenAlexVerifier("")
	v.BaseURL = srv.URL

	ref := Reference{DOI: "10.1038/nature12373"}
	res, err := v.Verify(context.Background(), ref)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if res.Status != VerifyError {
		t.Errorf("500 应视为 VerifyError，实际 %v", res.Status)
	}
}

func TestOpenAlex_Name(t *testing.T) {
	v := NewOpenAlexVerifier("")
	if v.Name() != "OpenAlex" {
		t.Errorf("Name 应为 OpenAlex，实际: %s", v.Name())
	}
}

func TestDetectLang(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Digital Governance", "en"},
		{"数字治理", "zh"},
		{"AI 与 数字治理", "zh"},
		{"", "en"},
	}
	for _, tt := range tests {
		if got := detectLang(tt.in); got != tt.want {
			t.Errorf("detectLang(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMapOpenAlexType(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"article", "article-journal"},
		{"book", "book"},
		{"book-chapter", "chapter"},
		{"dissertation", "thesis"},
		{"report", "report"},
		{"unknown-type", "article-journal"},
	}
	for _, tt := range tests {
		if got := mapOpenAlexType(tt.in); got != tt.want {
			t.Errorf("mapOpenAlexType(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
