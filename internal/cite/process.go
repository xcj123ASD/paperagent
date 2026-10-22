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