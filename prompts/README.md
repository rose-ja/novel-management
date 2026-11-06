# 阶段提示词索引

先阅读 `workflow/00-operating-manual.md`，再按阶段调用本目录提示词。提示词中的 `{{...}}` 必须替换成真实内容；不适用写 N/A，不得留空后要求 Agent 猜测。

## 项目级 P0～P8

| 阶段 | 提示词 | 主要输出 |
|---|---|---|
| P0 产品探索 | `P0-product-discovery.md` | product-brief、mvp-scope、research |
| P1 资料整理 | `P1-source-index.md` | sources、project-context |
| P2 需求定义 | `P2-requirements.md` | requirements、user-stories、glossary；必要时同步上游文档 |
| P3 UX/UI 设计 | `P3-ux-design.md` | user-flow、design-system、page-specs |
| P4 架构设计 | `P4-architecture.md` | architecture、tech-stack、API、data、testing、deployment、ADR |
| P5 红队审查 | `P5-red-team.md` | reviews/P5-red-team-review.md |
| P5b 修订复审 | `P5b-review-resolution.md` | 修订后的权威文档、审查结论 |
| P6 Spec/路线图 | `P6-spec-and-roadmap.md` | specs、roadmap |
| P7 工程初始化 | `P7-bootstrap.md` | 工程、AGENTS、bootstrap-verification |
| P8 启动验收 | `P8-bootstrap-verification.md` | bootstrap-verification |

## 变更级 F1～F10

| 阶段 | 提示词 | 主要输出 |
|---|---|---|
| F1 拆解 | `F1-task-breakdown.md` | proposal、design、tasks、checklist |
| F2 实现 | `F2-implementation.md` | 代码 Diff |
| F3 自测 | `F3-self-test.md` | verification |
| F4 独立审查 | `F4-code-review.md` | review findings |
| F5 修复复审 | `F5-fix-and-rereview.md` | 修复记录、复审结论 |
| F6 集成/E2E | `F6-e2e.md` | verification 中的 E2E 证据 |
| F7 人工验收 | `F7-human-acceptance.md` | acceptance |
| F8 PR/MR | `F8-pr.md` | PR/MR 描述 |
| F9 发布 | `F9-release.md` | 发布记录、Smoke Test |
| F10 归档 | `F10-archive.md` | 更新后的 Spec、归档目录 |

## 通用规则

`00-prompting-rules.md` 是所有阶段提示词的共同约束。特殊情况按需调用 `S1-change-impact.md`、`S2-architecture-issue.md`、`S3-project-retrospective.md`。
