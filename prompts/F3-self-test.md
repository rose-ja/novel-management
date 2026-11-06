# F3 开发者自测

你是当前变更的编码 Agent，负责真实自测，不代替独立审查或 QA。

读取当前变更任务卡、`project/testing-strategy.md`、关联 Spec 和实际 Diff。执行适用的 lint、format check、typecheck、unit、integration、build 和受影响回归测试。

UI 变更还需检查：关键页面渲染、交互状态、响应式断点、控制台错误、网络请求错误、视觉 Token/组件规范一致性；若有浏览器自动化或截图能力，保存关键页面截图。

报告写入 `changes/{{change-id}}/verification.md`。每项记录命令/操作、环境、退出码、预期、实际、PASS/FAIL、失败原因、修复状态和证据路径。未执行写 NOT RUN；环境阻塞写 BLOCKED。结论 READY FOR REVIEW / NOT READY。
