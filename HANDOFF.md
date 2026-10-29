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

### A5 断点续跑（2026-10-29 完成）
- commit 37ba583 + 43ac422：progress.json 每节完成后保存，中断后按 section name 合并恢复；BOM 自动剥离
- 新增 4 函数：loadProgress/saveProgress/removeProgress/processSection
- 测试：13 个 step3 测试全 PASS（含 BOM 场景）
- 实跑验证：手动造进度文件，引言保留"上次中断前的引言内容"，未重写
- 用法：默认启用，--resume=false 禁用；进度文件 <output>.progress.json，完成后自动删除

### A 组可靠性 5/5 天全部完成
- A1 字数控制 ✅（0e720ed + 87fb477）
- A2 质量检测 ✅（1d9c5ec）
- A3 20 次统计 ✅（acb873c，95% 通过率）
- A4 端到端测试 ✅（baf909b，7 个 cmd/paper 测试）
- A5 断点续跑 ✅（37ba583 + 43ac422）
- 下一组：B 分发安装（B1 单 exe、B2 install.ps1、B3 首启引导、B4 卸载），5 天

### B 组分发安装 5/5 天完成（2026-10-29）
- B1 `--install-pandoc`：多镜像（ghfast.top / gh-proxy.com / github.com），15 分钟超时，失败给 3 条手动方案
- B2 `scripts/install.ps1`：检测 Go/Pandoc/API Key，打印运行方式
- B3 首启引导：无 Key 打印 DeepSeek/智谱申请地址和 setx 命令
- B4 `scripts/uninstall.ps1`：清理 ~/.paperagent/ + 5 个环境变量，需 y 确认
- 新增包：`internal/pandocinstall`（下载/解压/校验/多镜像）
- **铁律**：所有 .ps1 写完必须重写 BOM，否则 PowerShell 5.1 按 GBK 读中文乱码
- 已知限制：本机网络受限，自动下载 pandoc 不可用，手动 winget 或离线 exe 兜底
- 下一组：C 成本配额（C1 token 统计、C2 单次上限、C3 数据外发告知、C4 成本预估），3 天

### C 组成本配额 3/3 天完成（2026-10-29）
- C1 token 统计：`client.go` 加 `UsageReporter` 接口、`OpenAICompatClient.Usage()`、`addUsage`；`main.go` 加 `reportUsage` 写 ~/.paperagent/usage.log
- C2 单次/单日上限：`--max-tokens-per-run`、`--max-tokens-per-day`，默认 0 不限；单日超限拒绝运行，单次超限保留结果打 TOKENS_OVER_LIMIT=1
- C3 数据外发告知：首次真实运行打印，写 ~/.paperagent/.data_notice_shown 标记
- C4 成本预估：跑前打印 ESTIMATED_TOKENS 和 ESTIMATED_COST，`--no-estimate` 可关
- 实测：一次 journal 跑约 11466 token，成本约 0.02 元
- 已知 LLM 幻觉：本次跑出 `照温忠麟(2014)`，refs 中无此文献，A2 检测正确抓到，需 prompt 约束加强或人工修正

### 累计进度
- A 可靠性 5/5 ✅
- B 分发安装 5/5 ✅
- C 成本配额 3/3 ✅
- 累计 13/62 天 ≈ 21%
- 下一组：D 用户体验（7 天），建议从 D2 错误可读开始，D1 本地 Web 3 天需先确认依赖

### D2 错误可读（2026-10-29 完成）
- 新建 `internal/errhint` 包：HTTP 401/402/403/404/429/5xx、超时、pandoc、design/refs 加载、token 配额、生成内容过短、润色失败，全部翻译成「原因 + 怎么办」
- 接入两处：`main()` 里 `errhint.Explain(err)` 处理 run() 返回的 error；`run()` 里遍历 `frame.RiskHints` 翻译技术错误
- 已知问题：DeepSeek Key 失效时 FallbackClient 静默切智谱 glm-5.3，推理模型返回空，用户看到「生成内容过短」。errhint 已提示「可能被切到智谱」，根治需 FallbackClient 记切换日志（后续任务）
- 工具铁律：改 .go 用 VSCode（code .\文件），不用 notepad 整文件粘贴，notepad 会吃 `})` 换行和 `""` 引号
- 下一项：D4 README_USER.md（1 天）

### D3 历史记录（2026-10-29 完成）
- 每次跑完存 `~/.paperagent/history/<YYYYMMDD-HHMMSS>/`，含 `draft.md` + `meta.json`（时间、主题、模式、token、质量、占位符数）
- 保留最近 20 次，超出自动删最旧（`pruneHistory`）
- 两个子命令：`--history` 列最近 10 次，`--history-show N` 把第 N 次 draft 复制为 `draft_history_N.md`
- 新增 6 函数：historyDir/historyEntry/saveHistory/pruneHistory/listHistory/showHistory
- 实测：Mock 模式跑 2 次，`--history` 显示 2 条，`--history-show 1` 复制成功

### 累计进度
- A 5/5 ✅ / B 5/5 ✅ / C 3/3 ✅ / D 3/7（D2/D3/D4）
- 累计 16/62 天 ≈ 26%
- D 组剩余：D1 本地 Web（3 天）、D5 FAQ（1 天）

### D5 FAQ（2026-10-29 完成）
- 新建 `FAQ.md`（6465 字节，6 章 28 小节）：工作流、文献库构建、质量检测解读、成本、常见误解、进阶
- 和 README_USER.md 分工：README 入门，FAQ 深度

### 累计进度
- A 5/5 ✅ / B 5/5 ✅ / C 3/3 ✅ / D 4/7（D2/D3/D4/D5）
- 累计 17/62 天 ≈ 27%
- D 组剩余：D1 本地 Web（3 天）

### F1 Key 审计（2026-10-29 完成）
- 审计结果：无硬编码 Key（`sk-` / `api_key=` / `Bearer xxx` 模式全无匹配）
- 审计结果：无敏感文件在 git（`.env` / `key.txt` / `load.ps1` / `.pem` / `.key` 全无）
- `.gitignore` 覆盖：`key.txt`、`env.ps1`、`load.ps1`、`*.log`、`*.progress.json`、`runs/`、`stats*.csv`
- 补齐缺口：`draft_history_*.md`（D3 产物）、`draft_web_*.md`（Web 产物）
- 下一项：F2 日志脱敏

### F2 日志脱敏（2026-10-29 完成）
- `internal/llm/client.go`：`HTTPError.Error()` 把 body 截断到 200 字符，超出加 `...(truncated)`
- `cmd/web/main.go`：design 加载日志从完整路径改为 `filepath.Base()`，不泄露用户名
- 新增 2 测试：`TestHTTPError_TruncatesLongBody`、`TestHTTPError_ShortBodyNotTruncated`
- 审计结论：`usage.log` 只记 token/时间/耗时，无敏感信息，不用改
- commit：93098b8（ahead 1，push 网络失败待推）
- 下一项：F3 输入校验

### F3 文件大小校验（2026-10-29 完成）
- 新增 `internal/validate` 包：`validate.FileSize(path, limitBytes)`，默认上限 10MB，检查存在/目录/超限
- 接入 `internal/design/loader.go`（`os.ReadFile` 前）、`internal/cite/process.go`（`ProcessRefs` 和 `LoadRefs`）
- `LoadRefs` 额外加 `len(refs) == 0` 报错
- 5 个测试全 PASS
- commit：82533cc（ahead 2，push 网络失败待推）
- 下一步：F4 依赖安全

### F4 依赖安全（2026-10-29 完成）
- Go 1.27.2 → 1.27.3，修 GO-2026-6809（crypto/x509，CVSS 6.8 中危，可达路径 internal/pandocinstall）
- GOPROXY 改 `https://goproxy.cn,direct`，GOSUMDB 改 `sum.golang.google.cn`（官方代理连不上）
- govulncheck 装在 `%GOPATH%\bin`，CI 用 `Join-Path (go env GOPATH)` 定位，不依赖 PATH
- `scripts/ci.ps1` 加 `govulncheck ./...` 阻断模式，基线 `No vulnerabilities found.`
- commit：a4e1f04（ahead 4，push 网络失败待推）
- 下一步：F5 免责声明

### 累计进度（2026-10-29）
- A 5/5 ✅ / B 5/5 ✅ / C 3/3 ✅ / D 7/7 ✅ / F 4/5（F1–F4 完成，待 F5 免责声明、F6 数据留存）
- 累计约 23/62 天 ≈ 37%
### F5 免责声明（2026-10-29 完成）
- 新增 `DISCLAIMER.md`：产出性质、人工核验义务、数据外发、责任限制、撤回方式
- 新增 `internal/notice` 包：`ShowOnce(markerName, lines)` 通用首启提示，C3 与 F5 共用
- `cmd/paper/main.go`：`printDataNotice` 改调 `notice.ShowOnce`；新增 `printDisclaimer`；`run()` 里 `fs.Parse` 后调用（Mock 跳过）
- `cmd/web/main.go`：`main()` 里 `flag.Parse` 后调 `notice.ShowOnce(".disclaimer_shown", ...)`
- `README_USER.md`：开头补「使用前必读」段，链到 DISCLAIMER.md
- 标记文件：`~/.paperagent/.disclaimer_shown`
- 验证：`go run .\cmd\paper --history` 首跑打印 8 行声明，标记生成；二跑不再出现
- commit：5ca8a9a（ahead 6，push 网络失败待推）
- 下一步：F6 数据留存

### 累计进度（2026-10-29 晚）
- A 5/5 ✅ / B 5/5 ✅ / C 3/3 ✅ / D 7/7 ✅ / F 5/6（F1–F5 完成，待 F6 数据留存）
- 累计约 23.5/62 天 ≈ 38%
### F6 数据留存（2026-10-29 完成）
- 新增 `--clean history|logs|all` 子命令 + `--yes` 跳过确认
- `runClean` 清理 `~/.paperagent/history`、`usage.log`、两个首启标记；`pandoc/` 不动
- README 补「本地数据留存与清理」节（含路径作用/能否删表格）
- FAQ 补「数据存在哪里？怎么清理？」
- commit：a87d2ae（ahead 8，push 网络失败待推）
- F 组 6/6 完成

### 累计进度（2026-10-29 收尾）
- A 5/5 ✅ / B 5/5 ✅ / C 3/3 ✅ / D 7/7 ✅ / F 6/6 ✅
- 累计约 24/62 天 ≈ 39%