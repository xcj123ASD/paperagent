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