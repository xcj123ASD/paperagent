# paperagent 交接

更新时间：2026-10-19
当前阶段：M1.5 起点

## 已完成
- M0 框架层 ✅（step1/step2 已随 step3 链路真实跑通）
- M1 正文生成 ✅（DeepSeek deepseek-chat 真实验证，commit 871db7c）

## 当前配置
- LLM_PROVIDER=deepseek
- 模型：deepseek-chat
- 测试：llm 12 / frame 21 / step3 8，全 PASS

## 下一步：M1.5
- A. 双模式最小改动（~40 行，5 文件，先做）
  - 关键：KeyPointRequest 也要加 Mode
- B. 参考文献格式化 internal/cite/（1–2 周，后做）

## 红线
- Key 只在 $env:，不进代码不进 git
- 终端只粘命令，不粘输出
- 重定向用 cmd /c "... > file"
## 十五、M1.5 完成记录（2026-10-20）

| 子步 | commit | 内容 |
|---|---|---|
| 第 1 步 | c54608b | Mode 类型 + Config/Frame/Builder/KeyPointRequest |
| 第 2 步 | 19cc424 | ExpandSectionRequest.Mode + journal prompt 分支 + 备用 provider 从 qwen 换为智谱 glm-5.3 |
| 第 3 步 | 9a2a316 | orchestrator 传 cfg.Mode；cmd/step3 支持 --mode thesis/journal |

### 当前可跑
- `go run .\cmd\step3 --mode thesis` → thesis 初稿
- `go run .\cmd\step3 --mode journal` → journal 初稿（prompt 强调理论对话/批判性/不堆砌）

### 双模式生效验证
对比两份 draft.json 的 content 风格差异，journal 应更强调理论对话，thesis 更平实。

### 下一步
M2 参考文献格式化：internal/cite/ 包，先做 GB/T 7714 格式化（纯本地，无 API），再接 OpenAlex 单向校验。
### 双模式验证结果（2026-10-20）

| 验证项 | 结果 |
|---|---|
| journal / thesis 均产出真实中文 | ✅ |
| 无 Mock 输出 | ✅ |
| 语气差异可辨识 | ✅ journal 更学术（"张力""生成性/概率性"），thesis 更平实 |
| 理论对话（引用既有研究） | ❌ 两边都无 |
| 结论 | 机制生效，效果受限。理论对话需等 M2 真实文献，prompt 层无法单独解决 |

**已知限制**：journal prompt 里"理论对话"与"不虚构文献"互相制约，模型选择保守不引用。M2 接入真实文献后，把文献摘要喂入 prompt 才能真正实现理论对话。

**M1.5 结论**：机制验证通过，双模式可辨识。不因"理论对话缺失"返工，该缺口归属 M2。
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

### 对后续的影响
- M2.5 导出可以继续做，导出的是"带占位符的初稿"
- M3（研究设计辅助 + Stata）优先级应高于原计划

### 附带修复
- 英文作者格式化：支持 OpenAlex 的"Surname Initials"格式（Goldfarb A → Goldfarb A）
## 十八、自用测试记录（2026-10-20）

### 测试命令
go run .\cmd\paper --topic "数字化转型对制造业企业创新绩效的影响" --mode journal --refs refs_real.json --verify --export docx

### 结果
- 链路完整跑通：md + docx 都生成
- 参考文献 5 条正常
- **研究设计 / 实证结果 随机出现 0 字**（两次跑，空的章节不同）

### 结论
- 不是代码 bug，是模型对"无真实数据写实证"的随机拒绝
- prompt 层已到顶（占位符约束 + needsData 分支 + 重试 + 兜底都试过）
- 继续调 prompt 收益接近零
- 根本解法：M3 接入真实数据，或用户手动补（兜底占位符已给出可执行建议）

### 对 M3 的启示
M3 的优先级应提高。研究设计辅助不是"锦上添花"，是补上当前工具最大缺口的必要模块。
## 二十、温度与模型实验结论（2026-10-20）

### 对比实验

| 配置 | 空的章节 |
|---|---|
| DeepSeek + temperature 0.7 | 3 节（文献综述、研究设计、实证结果） |
| DeepSeek + temperature 0.3 | 1 节（实证结果） |
| **智谱 glm-5.3 + temperature 0.3** | **0 节** ✅ |

### 结论（关键）
1. **temperature 是空白的主因**：0.7 → 0.3，空白从 3 节降到 1 节
2. **智谱比 DeepSeek 更配合中文实证论文写作**：同温度下 0 节 vs 1 节
3. 默认 provider 已从 deepseek 改为 zhipu（`LLM_PROVIDER=deepseek` 才用 DeepSeek 主）
4. `env.ps1` 的 `LLM_PROVIDER` 也改为 zhipu

### 对之前"M3.1 结论"的修正
之前记录"prompt 层已到极限"不准确——真正的瓶颈是 temperature 配置错误。
temperature 0.3 + 智谱 = 0 节空，不需要继续调 prompt。

### 后续注意
- 不要随意调高 temperature
- 如果换模型，先用 design.json + refs_real.json 跑一次对比空节数
## 二十一、B 方案验证：研究设计模板化（2026-10-20）

### 做法
- `design.DesignInput.RenderSection()`：用结构化输入直接拼出研究设计正文
- `step3` 对"研究设计"节跳过 LLM，直接用模板填充
- 其他章节仍走 LLM

### 结果
- 研究设计节：✅ 100% 稳定，所有 design.json 字段都用上（CSMAR、专利申请量、双向固定效应、变量符号）
- 文献综述：仍随机空（模型行为，非数据问题）
- 实证结果：仍空（确实缺真实回归系数，合理）

### 结论
- 方法类章节（研究设计）不需要 LLM 自由发挥，模板生成更准更稳
- 需要文献/数据的章节（文献综述、实证结果），模板解决不了
- 当前合理水平：1–2 节空，研究设计已稳定

### 后续可考虑的模板化
- 实证结果：需要用户提供真实系数后才能模板化
- 文献综述：需要真实文献列表，属于 M2 范畴