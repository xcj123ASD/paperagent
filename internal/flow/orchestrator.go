package flow

import (
"crypto/sha256"
"encoding/hex"
"encoding/json"
"os"

"paperagent/internal/config"
"paperagent/internal/frame"
"paperagent/internal/llm"
"paperagent/internal/model"
"paperagent/internal/pipeline"
"paperagent/internal/retriever"
"paperagent/internal/rule"
"paperagent/internal/step3"
"paperagent/internal/verify"
)

type Orchestrator struct {
Cfg     config.Config
Catalog []model.Journal
}

func New(cfg config.Config, catalog []model.Journal) *Orchestrator {
var llmClient llm.Client
if os.Getenv("LLM_PROVIDER") != "" {
llmClient = llm.NewFromEnv()
}
if llmClient == nil {
llmClient = llm.MockClient{}
}

pipeline.Register(step3.Stage{LLM: llmClient})
pipeline.Register(pipeline.Stub{Name_: "step4_design"})
pipeline.Register(frame.Stage{B: &frame.Builder{
Rules: frame.DefaultRules(),
LLM:   llmClient,
}})
return &Orchestrator{Cfg: cfg, Catalog: catalog}
}

func (o *Orchestrator) Run(inputJSON []byte) ([]byte, error) {
ctx := pipeline.NewContext(o.Cfg.Mock, o.Cfg.Strict)
var in map[string]interface{}
if err := json.Unmarshal(inputJSON, &in); err != nil {
return nil, err
}
hash := sha256.Sum256(inputJSON)
trace := hex.EncodeToString(hash[:8])
ctx.Logf("init", "trace="+trace)

topic := pickStr(in, "topic", "生成式人工智能赋能政务服务")
disc := pickStr(in, "discipline", "公共管理")
sub := pickStr(in, "sub", "数字治理")
kw := pickKw(in, "keywords", []string{"人工智能", "政务服务", "风险治理"})
artType := pickStr(in, "article_type", "实证型")

var checks []model.Check
for _, c := range new(rule.Engine).ValidateTopic(topic, kw) {
checks = append(checks, c)
}

cat := o.Catalog
if len(cat) == 0 {
cat = retriever.MockCatalog
}
journals := retriever.MatchJournals(topic, disc, sub, kw, cat,
retriever.DisciplineWeights{
Discipline: o.Cfg.Weights.Discipline,
Column:     o.Cfg.Weights.Column,
Quality:    o.Cfg.Weights.Quality,
Hotness:    o.Cfg.Weights.Hotness,
})
for _, c := range new(rule.Engine).ValidateJournals(journals, artType) {
checks = append(checks, c)
}

topics := retriever.GenerateTopics(disc, sub, o.Cfg.Mock)
papers := retriever.SamplePapers(o.Cfg.Mock)
policies := retriever.PolicySources(o.Cfg.Mock)
checks = append(checks, new(verify.Checker).Check(papers, policies)...)

out := model.Step1Output{
Discipline:   disc,
Topic:        topic,
Keywords:     kw,
ArticleType:  artType,
Journals:     take(journals, 10),
Topics:       topics,
SamplePapers: papers,
PolicySrcs:   policies,
Checks:       checks,
TraceID:      trace,
InputHash:    hex.EncodeToString(hash[:]),
}
return json.MarshalIndent(out, "", "  ")
}

func pickStr(m map[string]interface{}, k, def string) string {
if v, ok := m[k].(string); ok && v != "" {
return v
}
return def
}

func pickKw(m map[string]interface{}, k string, def []string) []string {
if a, ok := m[k].([]interface{}); ok {
var out []string
for _, v := range a {
if s, ok := v.(string); ok {
out = append(out, s)
}
}
if len(out) > 0 {
return out
}
}
return def
}

func take[T any](s []T, n int) []T {
if n <= 0 {
return []T{}
}
if len(s) <= n {
return s
}
return s[:n]
}
