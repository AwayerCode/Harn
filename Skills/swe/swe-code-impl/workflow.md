---
name: swe-code-impl
description: 旧代码实现入口，按模块职责选择专业实现能力。
---

# 代码实现兼容入口

1. 业务能力与流程编排使用[业务实现](../swe-business-impl/workflow.md)。
2. 存储、缓存、消息与外部服务适配使用[基础设施实现](../swe-infrastructure-impl/workflow.md)。
3. 对象组装、启动关闭、构建与部署使用[运行组装与部署](../swe-runtime-impl/workflow.md)。
4. 界面使用[UI 实施](../swe-ui-impl/workflow.md)。

先确认本次模块、契约、适用标准与可写范围；跨层任务由 SWE 入口协调。此入口不另行维护通用实现规则。旧格式校验命令继续引用共同 schema。
