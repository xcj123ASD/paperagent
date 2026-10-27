# paperagent 项目交接文档（2026-10-24 更新）

> **本文档目标**：让任何一个新对话窗口读完就能接管，不需要回溯历史。
> **阅读顺序**：第零节 → 第一节 → 第二节 → 第五节（当前状态）→ 第六节（待办）

---

# 零、接手第一件事（严格按顺序）

## 第 1 步：加载环境

```powershell
cd E:\lunwenagent; . .\load.ps1```
**预期输出**：
```
loaded: deepseek=35 zhipu=49 pandoc=E:\tools\pandoc\pandoc-3.12.1\pandoc.exe
```
**报"禁止运行脚本"**：
```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass; cd E:\lunwenagent; . .\load.ps1
```

## 第 2 步：本地 CI
```powershell
cd E:\lunwenagent; .\scripts\ci.ps1
```

**预期最后一行**：`ALL PASSED`

## 第 3 步：git 状态
```powershell
cd E:\lunwenagent; git status
```

**预期**：`nothing to commit, working tree clean` + `up to date with 'origin/master'`。

## 第 4 步：读本文档第五节（当前状态），按第六节从第 1 项开始

---

# 一、项目定位（不可动摇）

**学术写作 Copilot，不是"AI 论文生成器"。**

- 目标用户：高校教师、博士生、科研人员
- 核心原则：**给选项不给结论，给草稿不给终稿**
- 红线：**agent 不替用户做因果解读、不提炼贡献点、不模仿行文风格**——这三样是人的活

---

# 二、完成度

| 维度 | 完成度 | 说明 |
|---|---|---|
| 工程链路 | **约 95%** | 阶段一~三+五全部完成，50 篇真实文献跑通 |
| 可投稿质量 | **约 50%** | 依赖真实数据 + 人工改稿，agent 只能出底稿 |

---

# 三、环境配置（已持久化）

## User 级环境变量（新窗口自动读）

| 变量 | 长度 | 说明 |
|---|---|---|
| `DEEPSEEK_API_KEY` | 35 | **默认主 provider** |
| `ZAI_API_KEY` | 49 | 智谱备用 |
| `PANDOC_PATH` | 路径 | `E:\tools\pandoc\pandoc-3.12.1\pandoc.exe` |

## 关键参数（代码位置）

| 参数 | 值 | 位置 | 原因 |
|---|---|---|---|
| **`temperature`** | **0.3** | `internal/llm/client.go` 的 `chat` 函数 | **不要调高**，0.7 时空白章节从 1 节涨到 3 节 |
| **默认 provider** | **DeepSeek 主** | `NewFromEnv` | 智谱 glm-5.3 是思考模型，reasoning_tokens 吃 80% max_tokens |
| **DeepSeek 模型名** | **`deepseek-chat`** | `NewFromEnv` 的 `deepseek = NewOpenAICompat(...)` | **不要用 `deepseek-flash`**：它是推理模型，`reasoning_tokens` 会吃光 maxTokens，长 prompt 返回空或答非所问 |
| **`maxTokens` 上限** | **32768** | `doExpandSection` | 为 60+ 篇综述留空间 |
| **综述权重** | **3.5** | `internal/frame/builder.go` 的 `chapterWeight` | 综述从 2000→3846 字（44 篇时）；50 篇时约 3846 |
| **default 权重** | **1.3** | 同上 | 引言/结论各约 1429 字 |
| **`thinking` 参数** | **已删除** | `client.go` 的 `chat` | 新版智谱 glm-5.x 传任何值都 400 |

---

# 四、目录结构（关键文件）

```
refs_real.json           # 真实文献库 50 篇（commit 403836f）
design.json              # 研究设计输入
draft.md / draft_final.md # 产物 / 手改底稿
internal/llm/client.go   # LLM 客户端（deepseek-chat）
internal/step3/expand.go # step3 逐节扩写
internal/render/render.go # Markdown 渲染（剥离重复标题）
cmd/paper/main.go        # 主入口
```

---

# 五、2026-10-24 更新摘要（必读）

## 本次会话 7 个 commit（全部已推 origin/master）

| commit | 内容 |
|---|---|
| `5cf5e32` | 测试隔离 LLM_PROVIDER=mock，cmd/paper 不再打网络 |
| `dccdd7a` | refs 35 → 44 |
| `855809e` | RiskHint：引言/结论模板兜底显式记录 |
| `acc157d` | **deepseek-flash → deepseek-chat，修复 LLM 返回空** |
| `403836f` | **refs 44 → 50（3 英文 + 3 中文）** |
| `56db273` | render 剥离 Content 开头重复标题 |
| `a14ac8a` | RiskHint：综述回落骨架显式记录 |

## 三个已修问题（旧文档未记）

1. **deepseek-flash 是推理模型**，reasoning_tokens 吃光 maxTokens，长 prompt 返回空/答非所问 → 改 deepseek-chat
2. **测试未隔离 LLM_PROVIDER**，cmd/paper 真调 API 卡 90s → t.Setenv("LLM_PROVIDER","mock")
3. **LLM 在 Content 开头重复写标题**，渲染出两个同名 ## → render TrimPrefix("## "+sec.Name)

## 当前待办（不阻塞）

- **A**：核对 5 篇高风险 conclusion，等原文摘要
- **B**：4 篇未引用文献，已决定不修
- **C**：填 design.json 的 results，等真实 Stata 系数

## 2026-10-24 最新跑验证

- 50 篇加载，综述 4179 字论述式，引言 1941 字，结论 2167 字
- 风险提示仅 2 条字数偏差（引言 +36%、结论 +53%），无 0 字降级
- 6 个章节标题各出现 1 次，无重复

### A1 字数控制（2026-10-26 完成）
- commit 0e720ed：prompt 硬约束 + 1.25 倍预算截断
- 后续 commit：综述分支截断 + RiskHint 带丢弃字数
- 验收：5/5 达标，引言 -3.6%~+23.0%，综述 +4.9%~+20.9%，结论 +4.7%~+22.4%
- 已知问题：综述 LLM 波动大（3635~6372），截断兜底但丢内容，A3 统计时关注
- 下一项：A2 质量检测（2 天）

### A3 20 次成功率统计（2026-10-27 完成）
- commit acb873c：stats.ps1 统计脚本 + quality 检测支持英文全名/中文姓翻译
- 验收：19/20 PASS，95%，超过 90% 门禁
- 字数稳定：引言 1815-2208、综述 2947-4291、结论 1517-2206，均在截断上限内
- Run 11 FAIL 原因：LLM 幻觉 Malgue(1990)，应为 Levinthal(1990)，检测正确抓到
- 结论：A2 检测有效，幻觉可见不阻塞；A3 门禁通过
- 下一项：A4 cmd/paper mock 端到端测试进 CI
