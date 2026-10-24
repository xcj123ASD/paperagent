# paperagent 交接

更新时间：2026-10-22
当前阶段：工程纪律已补齐，待模板化（HANDOFF 第 23 节）

## 已完成
- M0 框架层 ✅（step1/step2 已随 step3 链路真实跑通）
- M1 正文生成 ✅（commit 871db7c）
- M1.5 双模式 ✅（commit 9a2a316）
- M2 参考文献格式化 + OpenAlex 校验 ✅（commit 7d0d34e）
- M2.3 串联 cmd/paper ✅（commit 9e61f36）
- M2.4 英文作者修复 ✅
- M2.5 Word/LaTeX 导出 ✅（commit 99c90d4）
- M3.1 研究设计模板 + --polish-design ✅（commit 7b44826）
- 工程修复 7 项 ✅（2026-10-22，commit 26e57c0 及之前）

## 当前配置
- LLM_PROVIDER：默认 DeepSeek 主，智谱备用（设 LLM_PROVIDER=zhipu 才切智谱主）
- temperature: 0.3（关键，不要调高）
- 本地验证：`. .\load.ps1` 后跑 `.\scripts\ci.ps1` → ALL PASSED

## 下一步：模板化（见第 23 节）
- 实证结果模板 + ResultsInput
- 文献综述模板 + 文献注入
- 引言/结论模板兜底

## 红线
- Key 只在 $env:，不进代码不进 git
- 终端只粘命令，不粘输出
- 重定向用 cmd /c "... > file"
- 大段内容用 notepad 写，不用 PowerShell heredoc

## 十五、M1.5 完成记录（2026-10-20）

| 子步 | commit | 内容 |
|---|---|---|
| 第 1 步 | c54608b | Mode 类型 + Config/Frame/Builder/KeyPointRequest |
| 第 2 步 | 19cc424 | ExpandSectionRequest.Mode + journal prompt 分支 + 备用 provider 从 qwen 换为智谱 glm-5.3 |
| 第 3 步 | 9a2a316 | orchestrator 传 cfg.Mode；cmd/step3 支持 --mode thesis/journal |

### 当前可跑
- `go run .\cmd\step3 --mode thesis` → thesis 初稿
- `go run .\cmd\step3 --mode journal` → journal 初稿

### 双模式生效验证
对比两份 draft.json 的 content 风格差异，journal 应更强调理论对话，thesis 更平实。

## 十六、B 验证结论（2026-10-20）

### 真实场景验证（主题：数字化转型对制造业企业创新绩效的影响）

| 章节 | 结果 | 说明 |
|---|---|---|
| 引言 / 文献综述 / 结论 | ✅ 正常生成 | 不依赖具体数据 |
| 实证结果 | ✅ 框架 + 占位符 | prompt 加固生效 |
| 研究设计 | ❌ 0 字，需人工补充 | 3 个主题均复现，系统性边界 |

### 结论
- 研究设计这节在无真实数据/变量定义输入时，LLM 系统性返回空，prompt 层无法解决
- 兜底占位符已把 KeyPoints 转为可执行建议，用户照填即可
- 根本解法：M3 的 internal/analysis/（Stata）或用户提供真实数据

### 附带修复
- 英文作者格式化：支持 OpenAlex 的 "Surname Initials" 格式（Goldfarb A → Goldfarb A）

## 十八、自用测试记录（2026-10-20）

### 测试命令
go run .\cmd\paper --topic "数字化转型对制造业企业创新绩效的影响" --mode journal --refs refs_real.json --verify --export docx

### 结果
- 链路完整跑通：md + docx 都生成
- 参考文献 5 条正常
- 研究设计 / 实证结果 随机出现 0 字（两次跑，空的章节不同）

### 结论
- 不是代码 bug，是模型对"无真实数据写实证"的随机拒绝
- prompt 层已到顶（占位符约束 + needsData 分支 + 重试 + 兜底都试过）
- 根本解法：M3 接入真实数据，或用户手动补

## 二十、温度与模型实验结论（2026-10-20）

### 对比实验

| 配置 | 空的章节 |
|---|---|
| DeepSeek + temperature 0.7 | 3 节（文献综述、研究设计、实证结果） |
| DeepSeek + temperature 0.3 | 1 节（实证结果） |
| 智谱 glm-5.3 + temperature 0.3 | 0 节 ✅ |

### 结论（关键）
1. temperature 是空白的主因：0.7 → 0.3，空白从 3 节降到 1 节
2. 智谱比 DeepSeek 更配合中文实证论文写作：同温度下 0 节 vs 1 节
3. 默认 provider 已从 deepseek 改为 zhipu（LLM_PROVIDER=deepseek 才用 DeepSeek 主）
4. env.ps1 的 LLM_PROVIDER 也改为 zhipu
> **注（后续修正）**：第 3、4 条为 2026-10-20 当日判断。后续实测发现智谱 glm-5.3 为思考模型，`reasoning_tokens` 会吃掉约 80% `max_tokens`，content 极短，不适合做主模型。**已切回 DeepSeek 主、智谱备用**（`LLM_PROVIDER=zhipu` 才切主）。详见第二十二节"已知边界"。


### 对之前"M3.1 结论"的修正
之前记录"prompt 层已到极限"不准确——真正的瓶颈是 temperature 配置错误。
temperature 0.3 + 智谱 = 0 节空，不需要继续调 prompt。

### 后续注意
- 不要随意调高 temperature
- 如果换模型，先用 design.json + refs_real.json 跑一次对比空节数

## 二十一、B 方案验证：研究设计模板化（2026-10-20）

### 做法
- design.DesignInput.RenderSection()：用结构化输入直接拼出研究设计正文
- step3 对"研究设计"节跳过 LLM，直接用模板填充
- 其他章节仍走 LLM

### 结果
- 研究设计节：✅ 100% 稳定，所有 design.json 字段都用上
- 文献综述：仍随机空（模型行为，非数据问题）
- 实证结果：仍空（确实缺真实回归系数，合理）

### 结论
- 方法类章节（研究设计）不需要 LLM 自由发挥，模板生成更准更稳
- 需要文献/数据的章节（文献综述、实证结果），模板解决不了
- 当前合理水平：1–2 节空，研究设计已稳定

## 二十二、2026-10-20 收工状态

### 已完成
- M1 / M1.5 / M2 / M2.3 / M2.4 / M2.5 / M3.1 全部闭环
- 研究设计节模板化，100% 稳定
- --polish-design 开关：默认模板，可选 LLM 润色
- temperature 0.3，默认 provider 切换实验完成

### 已知边界（诚实记录）
- 文献综述、实证结果随机空 1–2 节，占位符给出可执行建议
- 实证结果缺真实回归系数，需用户跑完数据后手填
- 智谱 glm-5.3 是思考模型，reasoning_tokens 吃掉大量 max_tokens，不适合做主模型（已切回 DeepSeek 为默认主模型）

### 环境变量（重要）
- User 级已设 ZAI_API_KEY、DEEPSEEK_API_KEY、PANDOC_PATH
- 新窗口标准开场：`cd E:\lunwenagent; . .\load.ps1`
- load.ps1 从 User 级注册表读 Key，已 gitignore

## 二十三、下次任务：补齐三类空白章节（模板化方案）

### 背景
当前 1–2 节随机空：文献综述与研究假设、实证结果（偶尔引言/结论）。
根因是模型对"缺文献/缺数据"的章节拒写，prompt 层已到极限。
解法与研究设计节一致：模板化 + 结构化输入。

### 任务 1：实证结果模板（半天）

**design.json 加可选字段：**
```json
"results": {
  "main_coef": "0.052",
  "main_se": "0.015",
  "significance": "1%",
  "direction": "正向",
  "robustness": ["替换被解释变量", "剔除疫情年份", "工具变量法"]
}