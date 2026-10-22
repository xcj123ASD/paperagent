package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"paperagent/internal/cite"
)

func main() {
	inputFlag := flag.String("input", "refs.json", "输入文献列表 JSON 文件")
	formatFlag := flag.String("format", "gbt7714", "输出格式：gbt7714 / apa")
	verifyFlag := flag.Bool("verify", false, "是否通过 OpenAlex 校验真实性")
	mailtoFlag := flag.String("mailto", "", "OpenAlex 礼貌池邮箱（可选，加速请求）")
	flag.Parse()

	data, err := os.ReadFile(*inputFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取 %s 失败: %v\n", *inputFlag, err)
		os.Exit(1)
	}

	var refs []cite.Reference
	if err := json.Unmarshal(data, &refs); err != nil {
		fmt.Fprintf(os.Stderr, "解析 JSON 失败: %v\n", err)
		os.Exit(1)
	}
	if len(refs) == 0 {
		fmt.Fprintln(os.Stderr, "文献列表为空")
		os.Exit(1)
	}

	if *verifyFlag {
		v := cite.NewOpenAlexVerifier(*mailtoFlag)
		verified := 0
		notFound := 0
		errors := 0
		for i := range refs {
			res, err := v.Verify(context.Background(), refs[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "[%d] verify error: %v\n", i+1, err)
				errors++
				continue
			}
			switch res.Status {
			case cite.VerifyFound:
				verified++
				// 只有高置信度（DOI 精确匹配）才补全用户数据；
				// 标题匹配（0.6）可能误匹配，只提示不覆盖
				if res.Matched != nil && res.Confidence >= 0.8 {
					mergeRef(&refs[i], *res.Matched)
				} else if res.Matched != nil {
					fmt.Fprintf(os.Stderr, "[%d] low-confidence match: %s (confidence %.2f)\n",
						i+1, res.Matched.Title, res.Confidence)
				}
			case cite.VerifyNotFound:
				notFound++
				fmt.Fprintf(os.Stderr, "[%d] not found: %s\n", i+1, res.Message)
			case cite.VerifyError:
				errors++
				fmt.Fprintf(os.Stderr, "[%d] verify error: %s\n", i+1, res.Message)
			}
		}
		fmt.Fprintf(os.Stderr, "verify done: found %d, notfound %d, errors %d\n", verified, notFound, errors)
	}

	f := pickFormatter(*formatFlag)
	out, err := f.Format(refs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "格式化失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(out)
}

func pickFormatter(name string) cite.Formatter {
	switch name {
	case "apa":
		// TODO: APA 实现待补，暂回落 GB/T 7714
		fmt.Fprintln(os.Stderr, "APA 尚未实现，回落 GB/T 7714")
		return cite.GBT7714{}
	default:
		return cite.GBT7714{}
	}
}

// mergeRef 用 OpenAlex 拉回的元数据补全用户输入里缺失的字段。
// 不覆盖用户已填的字段，只填空缺。
func mergeRef(dst *cite.Reference, src cite.Reference) {
	if dst.Title == "" {
		dst.Title = src.Title
	}
	if dst.Year == 0 {
		dst.Year = src.Year
	}
	if dst.Container == "" {
		dst.Container = src.Container
	}
	if dst.Volume == "" {
		dst.Volume = src.Volume
	}
	if dst.Issue == "" {
		dst.Issue = src.Issue
	}
	if dst.Pages == "" {
		dst.Pages = src.Pages
	}
	if dst.DOI == "" {
		dst.DOI = src.DOI
	}
	if len(dst.Authors) == 0 {
		dst.Authors = src.Authors
	}
	if dst.Lang == "" {
		dst.Lang = src.Lang
	}
	if dst.Type == "" {
		dst.Type = src.Type
	}
}
