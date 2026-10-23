package stata

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates/*.do.tmpl
var templateFS embed.FS

// Generate 把 StataRequest 渲染成 .do 文本。
// 不执行、不落盘，只返回字符串；调用方决定写文件还是先做静态检查。
//
// 安全：模板内没有自由命令区，所有 Stata 命令由模板固定，
// 用户输入只能填充变量名与路径字段（经 Validate 校验非空）。
func Generate(req StataRequest) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
	}

	tmplName := "templates/" + req.Model + ".do.tmpl"
	if _, err := templateFS.ReadFile(tmplName); err != nil {
		return "", fmt.Errorf("stata: 模型 %q 暂无模板（需在 templates/%s.do.tmpl 补齐）", req.Model, req.Model)
	}

	tmpl, err := template.ParseFS(templateFS, tmplName)
	if err != nil {
		return "", fmt.Errorf("stata: 解析模板 %s 失败: %w", tmplName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, req); err != nil {
		return "", fmt.Errorf("stata: 渲染模板 %s 失败: %w", tmplName, err)
	}
	return buf.String(), nil
}
