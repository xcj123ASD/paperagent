# paperagent 用户手册

学术写作 Copilot。给底稿，不给终稿。

---

## 这是什么

把「论文主题 + 研究设计 + 参考文献」变成一篇结构完整的中文论文底稿（Markdown / docx）。

**它做**：章节结构、字数分配、文献综述论述、引言/结论草稿、GB/T 7714 参考文献格式。

**它不做**：替你编数据、替你做因果解读、替你提炼贡献点、替你模仿行文风格。**底稿必须人工改。**

---

## 装什么

### 必须

1. **Windows 10/11**
2. **Go 1.21+**（仅源码运行需要）：https://go.dev/dl/
3. **LLM API Key**（二选一）：
   - DeepSeek（推荐，便宜）：https://platform.deepseek.com/
   - 智谱（备用）：https://open.bigmodel.cn/

### 可选

4. **Pandoc**（导出 docx/tex 需要）：`winget install JohnMacFarlane.Pandoc`
---

## 三步跑通

### 1. 配置 Key（只需一次）

    setx DEEPSEEK_API_KEY "sk-你的Key"

关闭终端重开，或运行：

    cd E:\lunwenagent; . .\load.ps1

### 2. 检查环境

    cd E:\lunwenagent; .\scripts\install.ps1

看到「Go 已安装 / Pandoc 已配置 / API Key 已配置」即可。

### 3. 跑第一篇

    cd E:\lunwenagent
    go run .\cmd\paper --design design.json --mode journal --refs refs_real.json --export docx

约 1 分钟，消耗约 1 万 token（约 0.02 元）。产物 draft.md 和 draft.docx。

---

## 第一次跑会看到什么

1. **数据外发告知**（只显示一次）：说明哪些内容会发给 LLM
2. **成本预估**：ESTIMATED_TOKENS=12250-22750
3. **进度日志**：step1 → step2 → step3 各章节
4. **质量检测**：综述是否论述式、引用是否真实、占位符数量
5. **token 统计**：本次用量，写入 ~/.paperagent/usage.log

---

## 输入文件说明

### design.json（研究设计，必填）

字段：topic、data_source、sample、dependent_var、core_var、model。

### refs_real.json（参考文献库，可选但强烈建议）

每条含：authors、title、container、year、abstract、conclusion、method、relation。

**conclusion 字段是综述质量的关键**——LLM 靠它写「谁发现了什么」。空着综述会干瘪。

---

## 常用参数

- --topic：论文主题（覆盖 design）
- --mode：thesis 或 journal
- --refs：参考文献 JSON 路径
- --design：研究设计 JSON 路径
- --export：docx / tex / html
- --polish-design：研究设计节用 LLM 润色
- --max-tokens-per-day：单日 token 上限，0 不限
- --no-estimate：跳过成本预估
- --resume=false：禁用断点续跑
- --install-pandoc：自动装 pandoc（独立子命令）

---

## 产物怎么用

draft.md 是**底稿**，不是终稿。建议流程：

1. 通读全文，标记「读起来不对」的段落
2. 搜「此处需补充」，全部替换成你自己的内容
3. 查「风险提示」节，逐条处理
4. 核参考文献编号是否真实、格式是否对
5. 实证结果节全是占位符，必须填入你跑出的真实系数
6. 引言/结论大概率超预算，删冗余

---

## 常见问题

### 第一次跑提示「未检测到 API Key」

配置 Key：setx DEEPSEEK_API_KEY "sk-你的Key"，**关闭终端重开**。环境变量不会在当前终端自动生效。

### HTTP 402：余额不足

DeepSeek 欠费。去 https://platform.deepseek.com/ 充值，或临时切智谱：setx LLM_PROVIDER zhipu

### HTTP 401：Key 无效

检查 Key 是否完整（35 位）、是否过期。echo $env:DEEPSEEK_API_KEY 应该有输出。

### 跑得慢、超时

50 篇文献 prompt 长，单次 90 秒超时。减少 --refs 数量，或分多次跑（断点续跑会自动接上）。

### 导出 docx 失败：未找到 pandoc

运行 paperagent --install-pandoc，或 winget install JohnMacFarlane.Pandoc

### 综述是列表式骨架，不是论述

LLM 综述失败回落骨架。看「风险提示」节，那里会写「LLM 润色失败或过短」。重跑一次，或检查 Key/账户余额。

### 生成内容过短（0 字）

可能原因：1) Key 失效，FallbackClient 切到了智谱 glm-5.3（推理模型，易返回空）；2) DeepSeek 欠费；3) prompt 太长。临时 unset ZAI：$env:ZAI_API_KEY=""，强制走 DeepSeek。

### 断点续跑怎么用

默认启用。跑一半中断（Ctrl+C、断网、崩溃），重新跑同一条命令，已完成的章节跳过，从断点继续。进度文件 draft.md.progress.json，全部完成后自动删除。想从头跑：--resume=false，或删 draft.md.progress.json。

### 成本是多少

一次 journal 模式（5 章节、50 篇文献）约 1 万 token，约 0.02 元。DeepSeek 定价：input 1 元/百万 token，output 2 元/百万 token（以官网为准）。设上限防超支：--max-tokens-per-day 100000

### 数据会不会泄露

以下内容会发给 LLM 服务商（默认 DeepSeek）：design.json 中的主题、变量、模型、数据来源描述；refs_real.json 中的标题、作者、年份、摘要、结论；章节标题与要点。**不会发送**：CSMAR 原始 .dta 文件、draft.md 已有正文、API Key。如有未发表数据或敏感信息，**先脱敏再跑**。

---

## 卸载

运行 .\scripts\uninstall.ps1，清理 ~/.paperagent/ 目录和 5 个环境变量（DEEPSEEK_API_KEY、ZAI_API_KEY、ZHIPU_API_KEY、PANDOC_PATH、LLM_PROVIDER）。源码目录需手动删除。

---

## 已知边界

- 实证结果节：永远是占位符，必须填入真实 Stata 系数
- 引言/结论超预算：常见 +30%~50%，需人工删减
- LLM 偶发幻觉：引用 refs 外作者，风险提示会标注
- 综述偶发回落骨架：LLM 失败时，看风险提示重跑
- 参考文献摘要字段：空着综述会干瘪，建议补全
- 不支持 macOS：当前仅 Windows

---

## 更多

- 开发交接文档：HANDOFF.md
- 本地 CI：.\scripts\ci.ps1
- 安装检查：.\scripts\install.ps1
