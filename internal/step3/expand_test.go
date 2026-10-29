package step3

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"paperagent/internal/design"
	"paperagent/internal/llm"
	"paperagent/internal/model"
	"paperagent/internal/pipeline"
)

func makeFrameJSON(t *testing.T, f model.Frame) []byte {
	t.Helper()
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	return b
}

func simpleFrame() model.Frame {
	return model.Frame{
		Title:       "基层治理数字化",
		ArticleType: "实证型",
		WordCount:   2000,
		Sections: []model.Section{
			{Name: "引言", Role: "提出问题与研究意义", WordBudget: 1000, KeyPoints: []string{"背景", "意义"}},
			{Name: "文献综述", Role: "梳理研究缺口", WordBudget: 1000, KeyPoints: []string{"国内", "国外"}},
		},
	}
}

// introConclFrame 构造含"引言"与"结论"两节的 Frame，专门测试模板兜底。
// 不含"文献综述"节，避免综述分支干扰。
func introConclFrame() model.Frame {
	return model.Frame{
		Title:       "基层治理数字化",
		ArticleType: "实证型",
		WordCount:   2000,
		Sections: []model.Section{
			{Name: "引言", Role: "提出问题与研究意义", WordBudget: 1000, KeyPoints: []string{"背景", "意义"}},
			{Name: "结论与政策启示", Role: "总结发现与政策建议", WordBudget: 1000, KeyPoints: []string{"发现", "政策"}},
		},
	}
}

type failExpandClient struct{}

func (failExpandClient) KeyPoints(ctx context.Context, req llm.KeyPointRequest) ([]string, error) {
	return []string{"x"}, nil
}

func (failExpandClient) ExpandSection(ctx context.Context, req llm.ExpandSectionRequest) (string, error) {
	return "", errors.New("mock expand fail")
}

// shortExpandClient 返回成功但 <100 字的内容，触发"过短→重试→仍过短"路径。
type shortExpandClient struct{}

func (shortExpandClient) KeyPoints(ctx context.Context, req llm.KeyPointRequest) ([]string, error) {
	return []string{"x"}, nil
}

func (shortExpandClient) ExpandSection(ctx context.Context, req llm.ExpandSectionRequest) (string, error) {
	return "太短", nil
}

type recordExpandClient struct {
	reqs []llm.ExpandSectionRequest
}

func (r *recordExpandClient) KeyPoints(ctx context.Context, req llm.KeyPointRequest) ([]string, error) {
	return []string{"x"}, nil
}

func (r *recordExpandClient) ExpandSection(ctx context.Context, req llm.ExpandSectionRequest) (string, error) {
	r.reqs = append(r.reqs, req)
	// 返回足够长的内容（>100 字），避免触发 step3 的空内容重试逻辑
	return req.Section + "-content " + strings.Repeat("这是一段用于测试的模拟正文内容。", 10), nil
}

func TestStage_ExpandAllSections(t *testing.T) {
	input := makeFrameJSON(t, simpleFrame())
	ctx := pipeline.NewContext(true, false)
	out, err := Stage{LLM: llm.MockClient{}}.Run(ctx, input)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	var got model.Frame
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("解析输出失败: %v", err)
	}
	for _, s := range got.Sections {
		if strings.TrimSpace(s.Content) == "" {
			t.Errorf("章节 %q 的 Content 未填充", s.Name)
		}
	}
}

func TestStage_NilLLM_Skip(t *testing.T) {
	input := makeFrameJSON(t, simpleFrame())
	ctx := pipeline.NewContext(true, false)
	out, err := Stage{LLM: nil}.Run(ctx, input)
	if err != nil {
		t.Fatalf("nil LLM 不应报错: %v", err)
	}
	var got model.Frame
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("解析输出失败: %v", err)
	}
	for _, s := range got.Sections {
		if s.Content != "" {
			t.Errorf("nil LLM 时章节 %q 的 Content 应为空", s.Name)
		}
	}
}

func TestStage_EmptySections(t *testing.T) {
	f := model.Frame{Title: "x"}
	input := makeFrameJSON(t, f)
	ctx := pipeline.NewContext(true, false)
	if _, err := (Stage{LLM: llm.MockClient{}}).Run(ctx, input); err == nil {
		t.Fatal("sections 为空应报错")
	}
}

func TestStage_InvalidJSON(t *testing.T) {
	ctx := pipeline.NewContext(true, false)
	if _, err := (Stage{LLM: llm.MockClient{}}).Run(ctx, []byte("not json")); err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}

func TestStage_LLMFailure_AddsRiskHint(t *testing.T) {
	input := makeFrameJSON(t, simpleFrame())
	ctx := pipeline.NewContext(true, false)
	out, err := Stage{LLM: failExpandClient{}}.Run(ctx, input)
	if err != nil {
		t.Fatalf("不应整体报错: %v", err)
	}
	var got model.Frame
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("解析输出失败: %v", err)
	}
	if len(got.RiskHints) == 0 {
		t.Fatal("LLM 失败应产生 RiskHints")
	}
	for _, s := range got.Sections {
		if isLitReviewSection(s.Name) {
			if strings.TrimSpace(s.Content) == "" {
				t.Errorf("文献综述类章节 %q 应由骨架兜底，不应为空", s.Name)
			}
			continue
		}
		if s.Content != "" {
			t.Errorf("失败时章节 %q 的 Content 应为空", s.Name)
		}
	}
}

func TestStage_SkipsExistingContent(t *testing.T) {
	f := simpleFrame()
	f.Sections[0].Content = "已有正文，不重新生成"
	input := makeFrameJSON(t, f)
	rec := &recordExpandClient{}
	ctx := pipeline.NewContext(true, false)
	out, err := Stage{LLM: rec}.Run(ctx, input)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	var got model.Frame
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("解析输出失败: %v", err)
	}
	if got.Sections[0].Content != "已有正文，不重新生成" {
		t.Errorf("已有 Content 被覆盖，实际: %q", got.Sections[0].Content)
	}
	if len(rec.reqs) != 1 {
		t.Errorf("应只调用 1 次 ExpandSection，实际 %d 次", len(rec.reqs))
	}
}

func TestStage_PreviousSummaryPassed(t *testing.T) {
	input := makeFrameJSON(t, simpleFrame())
	rec := &recordExpandClient{}
	ctx := pipeline.NewContext(true, false)
	if _, err := (Stage{LLM: rec}).Run(ctx, input); err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if len(rec.reqs) != 2 {
		t.Fatalf("应调用 2 次 ExpandSection，实际 %d", len(rec.reqs))
	}
	if rec.reqs[0].PreviousSummary != "" {
		t.Error("第 1 节 PreviousSummary 应为空")
	}
	if rec.reqs[1].PreviousSummary == "" {
		t.Error("第 2 节 PreviousSummary 不应为空")
	}
	if !strings.Contains(rec.reqs[1].PreviousSummary, "引言") {
		t.Errorf("第 2 节 PreviousSummary 应包含第 1 节名称，实际: %q", rec.reqs[1].PreviousSummary)
	}
}

func TestStage_ArticleTypePropagated(t *testing.T) {
	f := simpleFrame()
	f.ArticleType = "综述型"
	input := makeFrameJSON(t, f)
	rec := &recordExpandClient{}
	ctx := pipeline.NewContext(true, false)
	if _, err := (Stage{LLM: rec}).Run(ctx, input); err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	for _, r := range rec.reqs {
		if r.ArticleType != "综述型" {
			t.Errorf("ArticleType 未透传，got %q", r.ArticleType)
		}
	}
}

func TestStage_IntroTooShort_UseTemplate(t *testing.T) {
	d := &design.DesignInput{
		Topic: "测试主题",
		Model: "双向固定效应模型",
	}
	input := makeFrameJSON(t, introConclFrame())
	ctx := pipeline.NewContext(true, false)
	out, err := Stage{LLM: shortExpandClient{}, Design: d}.Run(ctx, input)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	var got model.Frame
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("解析输出失败: %v", err)
	}
	var intro model.Section
	for _, s := range got.Sections {
		if s.Name == "引言" {
			intro = s
		}
	}
	if strings.Contains(intro.Content, "本节内容生成不足") {
		t.Errorf("引言过短时应走模板兜底而非占位符，实际: %s", intro.Content)
	}
	if !strings.Contains(intro.Content, "本文余下部分安排如下") {
		t.Errorf("引言应含模板特征'本文余下部分安排如下'，实际: %s", intro.Content)
	}
}

func TestStage_ConclusionTooShort_UseTemplate(t *testing.T) {
	d := &design.DesignInput{
		Topic: "测试主题",
		Model: "双向固定效应模型",
	}
	input := makeFrameJSON(t, introConclFrame())
	ctx := pipeline.NewContext(true, false)
	out, err := Stage{LLM: shortExpandClient{}, Design: d}.Run(ctx, input)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	var got model.Frame
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("解析输出失败: %v", err)
	}
	var concl model.Section
	for _, s := range got.Sections {
		if s.Name == "结论与政策启示" {
			concl = s
		}
	}
	if strings.Contains(concl.Content, "本节内容生成不足") {
		t.Errorf("结论过短时应走模板兜底而非占位符，实际: %s", concl.Content)
	}
	if !strings.Contains(concl.Content, "本文仍存在一定局限") {
		t.Errorf("结论应含模板特征'本文仍存在一定局限'，实际: %s", concl.Content)
	}
}

// TestStage_ProgressSaveAndResume 验证进度文件恢复：已完成的节跳过，进度文件完成后删除。
func TestStage_ProgressSaveAndResume(t *testing.T) {
	dir := t.TempDir()
	progressPath := filepath.Join(dir, "draft.md.progress.json")

	pd := `{"sections":[{"name":"引言","content":"上次完成的引言内容"}]}`
	if err := os.WriteFile(progressPath, []byte(pd), 0644); err != nil {
		t.Fatalf("write progress: %v", err)
	}

	input := makeFrameJSON(t, simpleFrame())
	rec := &recordExpandClient{}
	ctx := pipeline.NewContext(true, false)
	out, err := Stage{LLM: rec, ProgressPath: progressPath}.Run(ctx, input)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	var got model.Frame
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("解析输出失败: %v", err)
	}
	if got.Sections[0].Content != "上次完成的引言内容" {
		t.Errorf("引言应保留进度内容，实际: %q", got.Sections[0].Content)
	}
	if len(rec.reqs) != 1 {
		t.Errorf("应只调 1 次 ExpandSection（仅文献综述），实际 %d", len(rec.reqs))
	}
	if _, err := os.Stat(progressPath); !os.IsNotExist(err) {
		t.Error("完成后进度文件应被删除")
	}
}

// TestStage_ProgressRoundTrip 验证 saveProgress/loadProgress 往返。
func TestStage_ProgressRoundTrip(t *testing.T) {
	dir := t.TempDir()
	progressPath := filepath.Join(dir, "draft.md.progress.json")

	src := simpleFrame()
	src.Sections[0].Content = "第一节正文"
	src.Sections[1].Content = ""
	s := Stage{ProgressPath: progressPath}
	s.saveProgress(&src)

	if _, err := os.Stat(progressPath); err != nil {
		t.Fatalf("进度文件应存在: %v", err)
	}

	dst := simpleFrame()
	s.loadProgress(&dst)
	if dst.Sections[0].Content != "第一节正文" {
		t.Errorf("loadProgress 应恢复第一节内容，实际: %q", dst.Sections[0].Content)
	}
	if dst.Sections[1].Content != "" {
		t.Errorf("第二节原为空，不应被恢复，实际: %q", dst.Sections[1].Content)
	}
}
