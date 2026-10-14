# AGENTS.md

## 提交信息规则

- 提交格式遵循 `docs/git-commit-rules.md`：`{type} : {breaking?} {subject}`，type 从 feat/fix/docs/style/refactor/test/chore/revert/enhance 中选择，subject 用简洁中文描述。
- 提交信息只描述本次修改了什么，**禁止出现 R1/R2/R3 等评审轮次编号或"第 X 轮"字样**。
- 每次提交只处理一个问题，不要把多个不相关的修改混在一起。

## 文档维护

- 较大模块功能更新、模块能力拓展时，需反向更新 `docs/system_design.md`（但不允许改动其中「AI 开发守则」一节的内容）。
- 测试增删时同步更新 `docs/unit_tests.md`（保持测试函数实名与数量一致）。
