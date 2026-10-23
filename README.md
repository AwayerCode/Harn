# Harn

让 Agent 的工作有方法、有边界、有验证。

Harn 是一套以 Skill 为核心的 Agent Harness：通过 Skill 连接 Agent 与方法、规则和工具，通过任务内外两个循环管理任务质量与任务推进。

## 两层架构

Agent 层负责理解任务、判断和执行；Harness 层以 Skill 为核心，组织方法流程、标准契约、CLI 和脚本等工具。

Skill 既是两层的关联，也是 Harness 内部的能力入口：Agent 读取 Skill，确定如何开展工作、何时调用工具，以及如何根据结果继续行动。

> Agent 层 ⇄ Skill ⇄ Harness 的方法、规则与工具

SWE 是 Harn 面向软件工程的能力，覆盖设计、实现、测试和审查。代码修改遵守任务范围并验证实际结果；代码审查报告问题的位置、影响和依据。

## 两个循环

任务外循环负责分发任务、接收结果，并决定下一项任务；任务内循环负责执行、校验、反馈和修正。

> 外循环：分发任务 → 任务内循环 → 接收结果 → 决定并分发下一任务
>
> 内循环：Agent 执行 → 工具校验 → 反馈问题 → Agent 修正 → 再校验

任务内满足完成条件后交付结果；受阻、停滞或达到约定上限时，返回当前状态与所需决策，由外循环决定后续行动。循环遵守任务职责：实现任务可以在授权范围内修正，只读审查以交付有依据的发现为完成，不自动进入修复。

## 稳定校验与持续改进

Skill 定义方法与流程，工具提供可执行的检查和反馈。校验通过只证明已检查的要求满足，Agent 仍需结合任务目标确认是否完成。

SWE 采用 Go CLI + Skill 的演进方案：稳定校验放入 Go 独立可执行文件 `swe-cli`，方法、流程与项目适配由 Skill、项目约定和配置承担。项目已有测试、lint 和构建由 Agent 按 Skill 调用，不设置 TS 扩展层。改进方法不能顺带降低验收标准。

当前仓库包含 Skills、Go 分发工具 `harn` 和 Go swe-cli 检查核心；knowledge-doc-guard 独立使用 Python 检查。现有 SWE 检查已迁入 Go，harn 已支持按清单安装配套工具；发行资产及跨平台运行验收需另行完成，目标方案以系统设计和接口协议为准；任务分发调度器不在本阶段范围。

开发版增加可选的 [YAML 模块架构](skills/swe/references/architecture/guide.md)：以层目录和模块目录落实设计，校验 Go 模块依赖并生成只读 HTML；需要新版配套检查器，尚待发布。

本阶段聚焦独立任务：闭环统一放在 SWE 的共享执行规范中，Agent 执行循环，swe-cli 提供单次校验。目标架构、模块边界及迁移安排见[系统设计](docs/system-design.md)，数据与失败语义见[接口协议](docs/interface-protocol.md)；两份文档描述待实施方案。

## 使用示例

项目内安装工具和 SWE 的命令（macOS / Linux；示例使用旧版 `v0.2.1` 的 `Skills/swe` 路径，新版源码使用 `skills/swe`）：

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh
.harness/bin/harn install --source https://github.com/AwayerCode/Harn --version tag:v0.2.1 --path Skills/swe
```

支持读取项目文件的 Agent 可通过以下任务指令使用 SWE：

> 先读取 `.harness/Skills/swe/SKILL.md`，选择代码审查方法，审查当前项目尚未提交的代码改动。只报告有依据的问题，说明文件位置、影响和原因，不修改文件。缺少必要依据时明确说明。

安装完成后，入口位于 `.harness/Skills/swe/SKILL.md`。项目接入、版本管理和更新说明见 [使用指南](HARN.md)。Windows 下载方式见 [安装](HARN.md#安装)。

## 许可证

[MIT](LICENSE)
