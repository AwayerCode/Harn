---
name: system-design
description: 从业务模型设计或审查分层、模块、文件结构、依赖与数据契约，维护系统设计及生成视图，不修改生产代码和测试。
---

# 系统设计

## 共同约定

本领域随 `swe` 统一安装和版本更新，使用包内的[领域边界](../references/standards/domains.md)、[执行规范](../references/standards/execution.md)和[CLI 约定](../references/cli.md)。按任务读取相关方法，不执行其他领域的完整流程。`contract.json` 只声明可引用的产物类型，不要求全部输入齐备，也不自动授权文件写入。

## 责任与范围

维护指定架构 YAML、设计理由、必要数据协议、技术与运行约束，以及由设计生成的 HTML。架构决定层、模块及文件归属；业务模型、生产代码、测试与实际部署配置只读。架构发现实现偏差时报告生产代码位置，不能移动或修复源码；发现业务模型缺口时反馈业务责任方。

## 方法与依据

按需读取[分层与技术选择](references/layers.md)、[模块与数据协议](references/modules-and-contracts.md)。责任层次不等于源码依赖方向；平级正交业务能力保持独立，由编排模块组合；业务通过抽象边界使用技术能力，具体对象由运行组装连接。不能以文件数量决定模块，也不能因 CLI 允许某条边就认定其设计合理。

先确认目标交付环境、用户可用条件和业务状态所有权，再设计必要的层与模块。设计层映射到层目录，模块映射到层内独立目录，文件承载模块内能力；内部结构留有实现自主性，不能增加无归属源码。按成本、风险与控制权确定自研、复用或外部集成。

## 表达与检查

采用 YAML 时使用共享[模块架构规范](../references/architecture/guide.md)与 Schema，结构关系只维护一份。层设计可用 `architecture validate --profile architecture` 交接；模块齐备后运行完整 `architecture validate`，需要视图时运行 `architecture render`。已有源码的对照使用 `architecture check`，不因此获得源码写入权。

CLI 当前只覆盖模块结构与 Go 源码导入；设计理由、状态所有权、数据语义、失败、幂等、兼容和生命周期仍需专业判断。接口机器定义沿用已有权威来源，不复制字段；暂未实现接口检查不等于可以省略必要协议。

已有 Markdown 设计沿用[文档表达](references/markdown-design.md)、[系统设计模板](assets/system-design.md)和[协议模板](assets/interface-protocol.md)，使用 `documents`、`design` 与 `structure`。不为通过旧检查器再抄一份 YAML 架构表格。YAML 项目的独立协议按 `documents --kind interface-protocol` 检查，模块引用一致性人工核对。

## 完成条件

本次职责、文件结构和依赖关系清楚，重要协议与运行承诺有验证依据；实际检查与语义审查完成。层设计局部交付不能被声称为完整设计，设计通过不能被声称为实现通过。只读审查不修改 YAML 或重新生成视图。
