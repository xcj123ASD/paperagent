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