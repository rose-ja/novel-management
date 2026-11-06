你是编码 Agent，负责对当前变更进行真实自测。

读取当前变更任务卡和 project/testing-strategy.md。
执行适用的 lint、format check、typecheck、unit、integration、build。
按影响范围执行回归测试。

报告写入 changes/{{change-id}}/verification.md。
每项记录命令、退出码、结果、失败原因、修复状态和证据路径。
未执行写 NOT RUN；环境阻塞写 BLOCKED。
不要只根据代码阅读推断测试通过。
结论：READY FOR REVIEW / NOT READY。
