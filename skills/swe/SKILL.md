---
name: swe
description: 提供业务模型、系统设计、代码实现、测试与发布交付五个领域能力，包含项目接入、更新和退出能力，统一共享规范、Schema 和 CLI，按任务选用。
---

# SWE

SWE 以软件价值交付为目标。业务模型、系统设计、代码实现、测试和发布交付是五个领域能力，按任务选用，不是必须顺序执行的五个阶段。交付环境与验证条件可提前参与设计，缺陷反馈回对应产物所有者。

## 领域能力

| 能力 | 入口与责任 |
| --- | --- |
| 业务模型 | [business-model](business-model/SKILL.md)：实体、行为、流程与规则 |
| 系统设计 | [system-design](system-design/SKILL.md)：层、模块、文件结构、依赖与数据契约 |
| 代码实现 | [implementation](implementation/SKILL.md)：生产代码、技术适配与内部组装 |
| 测试 | [testing](testing/SKILL.md)：测试设计、单元／集成／端到端测试与证据 |
| 发布交付 | [delivery](delivery/SKILL.md)：发布构建、环境部署、运行验证与恢复 |

五个领域随 `skills/swe` 一次安装、统一更新和登记版本，共用一个 CLI。顶层入口按任务明确读取对应领域的 `SKILL.md`，不依赖宿主自动发现嵌套入口。各领域内完成设计、实现或审查；旧领域及交互入口已删除。

## 项目接入

已有或新项目接入 SWE、建立或修订已有项目 YAML、更新工具和退出时，读取 [adoption](adoption/SKILL.md)。它按需组织五个领域，不是第六个领域或固定流水线；随 SWE 统一安装和版本，共用同一个 CLI。

## 共享规则

首先按[领域边界](references/standards/domains.md)确定产物所有权，再按[加载规范](references/standards/loading.md)读取当前所需内容。范围和失败处理遵守[执行规范](references/standards/execution.md)，专业判断遵守[共同工程标准](references/standards/engineering.md)。

[CLI 约定](references/cli.md)定义检查选择与输出边界；[产物规范](references/standards/artifacts.md)维护文档表达，[YAML 模块架构](references/system-design/guide.md)定义结构化架构。共享方法、Schema 和 CLI 只维护一份，不复制进各领域或业务项目。

## 协作与改动范围

领域入口不授予全仓写入权。代码实现不维护任何测试，测试不修改生产代码和业务预期；各领域都可在授权范围内自检修正，也可只读审查。混合文件细化到条目，目录含其他领域文件时使用文件清单。跨领域缺口明确归属，不顺手修改。

共享能力维护按[维护方法](references/maintenance.md)同步规则、检查器与消费者。普通项目任务只读取共享资源，不修改它们来迁就失败结果。
