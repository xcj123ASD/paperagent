package main

import (
"context"
"flag"
"fmt"
"os"
"path/filepath"

"paperagent/internal/export"
)

func main() {
inputFlag := flag.String("input", "draft.md", "输入 Markdown 文件")
outputFlag := flag.String("output", "", "输出文件（扩展名决定格式：.docx / .tex / .html；为空时按输入名替换）")
flag.Parse()

out := *outputFlag
if out == "" {
base := filepath.Base(*inputFlag)
ext := filepath.Ext(base)
stem := base[:len(base)-len(ext)]
out = stem + ".docx"
}

e := export.NewPandocExporter()

fmt.Printf("输入: %s\n", *inputFlag)
fmt.Printf("输出: %s\n", out)

if err := e.Export(context.Background(), *inputFlag, out); err != nil {
fmt.Fprintf(os.Stderr, "导出失败: %v\n", err)
os.Exit(1)
}

fi, err := os.Stat(out)
if err != nil {
fmt.Fprintf(os.Stderr, "输出文件不存在: %v\n", err)
os.Exit(1)
}
fmt.Printf("导出成功: %s (%d 字节)\n", out, fi.Size())
}