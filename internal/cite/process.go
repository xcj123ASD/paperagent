package cite

import (
	"context"
	"encoding/json"
	"os"
)

// ProcessRefs 读取 refs.json，可选校验，按 GB/T 7714 格式化，返回参考文献文本。
func ProcessRefs(path string, verify bool, mailto string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var refs []Reference
	if err := json.Unmarshal(data, &refs); err != nil {
		return "", err
	}
	if len(refs) == 0 {
		return "", nil
	}
	if verify {
		v := NewOpenAlexVerifier(mailto)
		for i := range refs {
			res, err := v.Verify(context.Background(), refs[i])
			if err != nil {
				continue
			}
			if res.Status == VerifyFound && res.Matched != nil && res.Confidence >= 0.8 {
				Merge(&refs[i], *res.Matched)
			}
		}
	}
	f := GBT7714{}
	return f.Format(refs)
}

// LoadRefs 仅加载参考文献 JSON，不做 OpenAlex 校验与 GB/T 7714 格式化。
// 用于把原始 []Reference 结构传给 step3，供文献综述环节引用真实文献。
func LoadRefs(path string) ([]Reference, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var refs []Reference
	if err := json.Unmarshal(data, &refs); err != nil {
		return nil, err
	}
	return refs, nil
}
