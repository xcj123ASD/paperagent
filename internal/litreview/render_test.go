package litreview

import (
	"strings"
	"testing"

	"paperagent/internal/cite"
)

func TestRenderSkeletonEmpty(t *testing.T) {
	out := RenderSkeleton(nil)
	if strings.TrimSpace(out) == "" {
		t.Fatal("空 refs 不应返回空字符串")
	}
	if !strings.Contains(out, "此处需补充文献综述") {
		t.Errorf("空 refs 应出占位，实际：%s", out)
	}
}

func TestRenderSkeletonSingle(t *testing.T) {
	out := RenderSkeleton([]cite.Reference{{
		Authors: []string{"张三"},
		Title:   "测试文献",
		Year:    2023,
	}})
	for _, w := range []string{"张三", "2023", "测试文献"} {
		if !strings.Contains(out, w) {
			t.Errorf("缺少 %q，实际：%s", w, out)
		}
	}
}

func TestRenderSkeletonMultiYear(t *testing.T) {
	out := RenderSkeleton([]cite.Reference{
		{Authors: []string{"张三"}, Title: "A", Year: 2023},
		{Authors: []string{"李四"}, Title: "B", Year: 2021},
		{Authors: []string{"王五"}, Title: "C", Year: 2021},
	})
	for _, w := range []string{"张三", "李四", "王五", "2023 年", "2021 年"} {
		if !strings.Contains(out, w) {
			t.Errorf("缺少 %q，实际：%s", w, out)
		}
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("非空 refs 不应返回空")
	}
}
