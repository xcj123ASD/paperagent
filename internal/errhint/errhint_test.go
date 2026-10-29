package errhint

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"paperagent/internal/llm"
)

func TestExplain_Nil(t *testing.T) {
	if got := Explain(nil); got != "" {
		t.Errorf("nil 应返回空，实际: %q", got)
	}
}

func TestExplain_HTTP402(t *testing.T) {
	err := &llm.HTTPError{StatusCode: 402, Body: "insufficient balance"}
	hint := Explain(err)
	if !strings.Contains(hint, "充值") {
		t.Errorf("402 应提示充值，实际: %s", hint)
	}
}

func TestExplain_HTTP401(t *testing.T) {
	err := &llm.HTTPError{StatusCode: 401, Body: "unauthorized"}
	hint := Explain(err)
	if !strings.Contains(hint, "Key 无效") {
		t.Errorf("401 应提示 Key 无效，实际: %s", hint)
	}
}

func TestExplain_HTTP429(t *testing.T) {
	err := &llm.HTTPError{StatusCode: 429, Body: "rate limited"}
	hint := Explain(err)
	if !strings.Contains(hint, "限流") {
		t.Errorf("429 应提示限流，实际: %s", hint)
	}
}

func TestExplain_HTTP500(t *testing.T) {
	err := &llm.HTTPError{StatusCode: 500, Body: "internal error"}
	hint := Explain(err)
	if !strings.Contains(hint, "服务端故障") {
		t.Errorf("5xx 应提示服务端故障，实际: %s", hint)
	}
}

func TestExplain_HTTPWrapped(t *testing.T) {
	base := &llm.HTTPError{StatusCode: 402, Body: "insufficient"}
	wrapped := fmt.Errorf("step3 failed: %w", base)
	hint := Explain(wrapped)
	if !strings.Contains(hint, "充值") {
		t.Errorf("包装后的 402 应提示充值，实际: %s", hint)
	}
}

func TestExplain_DeadlineExceeded(t *testing.T) {
	hint := Explain(context.DeadlineExceeded)
	if !strings.Contains(hint, "超时") {
		t.Errorf("超时应提示超时，实际: %s", hint)
	}
}

func TestExplain_PandocNotFound(t *testing.T) {
	err := errors.New("未找到 pandoc：运行 paperagent --install-pandoc 自动下载")
	hint := Explain(err)
	if !strings.Contains(hint, "pandoc") {
		t.Errorf("应提示 pandoc 安装，实际: %s", hint)
	}
}

func TestExplain_TokenLimit(t *testing.T) {
	err := errors.New("今日 token 用量已达上限（12345 / 10000）")
	hint := Explain(err)
	if !strings.Contains(hint, "配额") {
		t.Errorf("应提示配额，实际: %s", hint)
	}
}

func TestExplain_Unknown(t *testing.T) {
	err := errors.New("some unknown error")
	hint := Explain(err)
	if hint != "" {
		t.Errorf("未知错误应返回空，实际: %q", hint)
	}
}

func TestExplain_ContentTooShort_Zero(t *testing.T) {
	err := errors.New("章节 \"引言\" 生成内容过短（0 字，预算 1782 字），需人工补充")
	hint := Explain(err)
	if !strings.Contains(hint, "空内容") {
		t.Errorf("0 字应提示空内容，实际: %s", hint)
	}
}

func TestExplain_ContentTooShort_NonZero(t *testing.T) {
	err := errors.New("章节 \"引言\" 生成内容过短（50 字，预算 1782 字），需人工补充")
	hint := Explain(err)
	if !strings.Contains(hint, "过短") {
		t.Errorf("非 0 字应提示过短，实际: %s", hint)
	}
}

func TestExplain_PolishFailed(t *testing.T) {
	err := errors.New("章节 \"文献综述\" LLM 润色失败或过短，已回落骨架")
	hint := Explain(err)
	if !strings.Contains(hint, "润色失败") {
		t.Errorf("应提示润色失败，实际: %s", hint)
	}
}
