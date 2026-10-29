# paperagent FAQ

README_USER.md 覆盖入门和常见错误。本文件补充工作流、文献库构建、质量检测解读、常见误解。

---

## 工作流

### 生成的底稿怎么用

三步：

1. **通读标记**：从头读一遍，用高亮标出「读起来不像人话」的段落
2. **处理占位符**：搜「此处需补充」，每处替换成你自己的内容；搜「本节内容生成不足」，整节重写
3. **核参考文献**：逐条对照 `refs_real.json`，确认 [N] 编号对应的文献真实存在、格式正确

底稿的价值是「结构 + 论述框架」，不是「能直接投的终稿」。

### thesis 还是 journal 模式

- **thesis**：学位论文，章节多、字数宽、允许铺陈
- **journal**：期刊论文，章节紧、字数严、要求论证密度

不确定就用 journal，写作标准更高，改起来更省事。

### 怎么对比多次跑的结果

每次跑完自动存 `~/.paperagent/history/<时间戳>/`，含 `draft.md` 和 `meta.json`。

    go run .\cmd\paper --history           # 列最近 10 次
    go run .\cmd\paper --history-show 2    # 把第 2 次的 draft 复制到当前目录

对比两次的综述段落，挑论述质量更好的那版。

---

## 文献库构建

### refs_real.json 怎么从零建

最少每条要有：

- authors（全名，英文用 `姓 名首字母` 格式，如 `Teece D J`）
- title
- container（期刊名）
- year
- lang（zh / en）

**强烈建议补**：abstract、conclusion、method、relation。

### conclusion 字段为什么关键

综述论述的质量取决于 `conclusion`。LLM 靠它写「谁发现了什么、谁和谁矛盾」。空着综述会退化成「作者(年份)研究了《标题》」的列表式骨架。

每条 `conclusion` 写一句话，例如：

    "数字化转型通过提升知识整合能力显著促进制造业企业创新绩效，效应在非国有企业中更强"

### 怎么加新文献

一次加 3–5 篇，不要一次加 40 篇。加完跑：

    go run .\cmd\cite --input refs_real.json --format gbt7714 | Measure-Object -Line

条数对、不报 `invalid character`，才算加成功。

### relation 字段怎么填

- **支持**：结论与本文假设方向一致
- **补充**：提供了额外机制或边界条件
- **反驳**：结论与本文假设相反（综述的「理论张力」靠这类文献）
- **理论借鉴**：提供理论框架，不直接检验
- **方法借鉴**：测度或识别方法可复用
- **方法局限**：方法本身有缺陷，需在综述里指出

---

## 质量检测解读

### QUALITY_STATUS=PASS 是不是就万事大吉

不是。PASS 只代表三件事：

1. 综述是论述式，不是骨架列表
2. 正文引用都在 refs 里，或已用占位符标注
3. 没有未处理的占位符

**不代表**：论证质量高、数据真实、结论正确、能直接投稿。

### WARN 和 FAIL 什么区别

- **FAIL**：裸引用幻觉（引用了 refs 外作者，且没加占位符提示），必须人工处理
- **WARN**：拼写变形、中文翻译变形、有提示的幻觉、占位符残留，可接受但需确认

WARN 不阻塞 PASS。

### 引用幻觉怎么办

分两种：

- **有提示的幻觉**：LLM 写了「刘诗源(2020)（此处需补充文献）」，自己有自知之明，按占位符处理
- **裸幻觉**：LLM 直接写「Malgue(1990)」，没有占位符，必须人工核对改成正确作者（如 `Levinthal`）

### 字数偏差多少算正常

引言/结论 ±30% 内可接受。超 30% 说明 LLM 不遵守字数约束，工具会截断并记风险提示。

综述 ±30% 内可接受，超了会被截断到 1.25 倍预算，丢弃的内容需人工补。

---

## 成本

### 一次跑多少钱

journal 模式（5 章节、50 篇文献）约 1 万 token，约 0.02 元。

DeepSeek 定价：input 1 元/百万 token，output 2 元/百万 token（以官网为准）。

### 怎么控制成本

    --max-tokens-per-day 100000   # 单日上限 10 万 token，超了拒绝运行
    --max-tokens-per-run 30000    # 单次上限 3 万 token，超了保留结果但警告

跑前会打印 `ESTIMATED_TOKENS` 和 `ESTIMATED_COST`。熟了之后加 `--no-estimate` 跳过。

### input 和 output 什么区别

- **input**：prompt 的 token 数（design + refs 列表 + 章节要点）
- **output**：LLM 生成的正文 token 数

50 篇文献时 input 约 6700，output 约 4500。input 占比高说明 refs 列表长。

---

## 常见误解

### 「AI 生成论文」

错。这是**学术写作 Copilot**，给底稿不给终稿。它不做因果解读、不提炼贡献点、不模仿你的行文风格——这三样是人的活。

### 「一键导出就能投稿」

错。导出的是**方法论完整、数据空缺**的底稿。实证结果节永远是占位符，必须填入你跑出的真实系数。

### 「自动找文献」

错。当前不支持自动抓取，`refs_real.json` 必须你自己维护。

### 「LLM 写的比我好」

不一定。LLM 擅长组织结构和铺陈论述，不擅长判断哪个结论更可信、哪个机制更重要。你的领域判断不可替代。

---

## 进阶

### 怎么自定义章节结构

编辑 `internal/frame/builder.go` 的 `DefaultRulesWithMode`，改 `DefaultStructure` 列表。加章节后 `go build` 重跑。

### 怎么调字数预算

编辑 `internal/frame/builder.go` 的 `chapterWeight`：

- 综述默认 3.5（50 篇时约 3465 字）
- 实证默认 1.8
- 研究设计默认 1.2
- 引言/结论走 default 1.8

权重总和变了，各章节预算按比例重新分配。

### 怎么换 LLM 模型

    setx LLM_PROVIDER zhipu   # 切智谱 glm-5.3

注意：智谱 glm-5.3 是推理模型，`reasoning_tokens` 会吃 60% maxTokens，长 prompt 容易返回空。默认 DeepSeek `deepseek-chat` 更稳。

### 怎么调试 LLM 输出

风险提示节会列出每个降级/截断/幻觉。想看更细的日志，查 `~/.paperagent/usage.log`（token 用量）和 `~/.paperagent/history/<时间戳>/meta.json`（每次运行的元数据）。

想看 LLM 实际收到什么 prompt，需改代码在 `internal/llm/client.go` 的 `chat` 函数加 `fmt.Println(prompt)`，仅调试用，不要提交。

### 团队多人怎么用

每人独立 Key、独立 `~/.paperagent/` 目录。不要共享 Key——一人跑爆额度全员停。

---

## 还有问题

先搜 `README_USER.md` 和本文件，找不到再提 issue。
