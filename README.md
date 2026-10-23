\# paperagent



\*\*学术写作 Copilot，不是“AI 论文生成器”。\*\*



面向高校教师、博士生与科研人员的写作助手：LLM 搭框架、写初稿，人定贡献、把关终稿。核心原则是\*\*给选项不给结论，给草稿不给终稿\*\*。



当前阶段：核心链路（正文生成 → 文献综述 → 研究设计 → 实证结果 → 引言/结论 → 参考文献 → 导出）已打通，模板化补齐，具备可用初稿生成能力。



\---



\## 环境要求



\- \*\*Go 1.22+\*\*

\- \*\*Pandoc\*\*（导出 docx/tex 需要；只生成 Markdown 可跳过）

\- \*\*LLM API Key\*\*：DeepSeek（默认主）或 智谱（备用）



\---



\## 快速开始



\### 1. 配置环境变量



在系统「用户环境变量」里设置（PowerShell）：



```powershell

\[Environment]::SetEnvironmentVariable("DEEPSEEK\_API\_KEY", "sk-xxx", "User")

\[Environment]::SetEnvironmentVariable("ZAI\_API\_KEY", "xxx", "User")            # 可选，智谱备用

\[Environment]::SetEnvironmentVariable("PANDOC\_PATH", "E:\\tools\\pandoc\\pandoc.exe", "User")

