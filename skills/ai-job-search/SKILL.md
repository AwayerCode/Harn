---
name: ai-job-search
description: 辅助求职定位、职位匹配评估、简历与求职信定制、面试准备和申请记录管理；基于 MadsLorentzen/ai-job-search 工作区，适用于个人求职流程。
---

# AI Job Search

本入口封装 MadsLorentzen/ai-job-search 的工作区模板。方法、职位门户工具、简历模板和命令规格完整保存在 `workspace/`，本文件仅负责工作区定位和宿主适配。

## 工作区

1. 已有求职工作区时，读取该工作区的 `AGENTS.md` 和配置，沿用其候选人资料与输出路径。
2. 首次使用时先确定用户的目标目录，将本 Skill 的 `workspace/` 内容复制到一个新的专用目录；不得覆盖现有文件。真实候选人资料、申请记录和生成文档只写入目标工作区，不写入本 Skill 的模板。
3. 从目标工作区读取 [上游指南](workspace/AGENTS.md) 对应的本地文件，按 [SETUP.md](workspace/SETUP.md) 配置资料和依赖。模板示例不代表用户履历；缺失信息向用户确认，不编造经历。
4. 所有上游相对路径和工具命令都以目标工作区为根执行。Python、Bun、LaTeX 以及可选 PDF 工具按任务需要检查；本次引入不安装运行依赖。

## 按任务加载

| 任务 | 模板中的权威入口 |
| --- | --- |
| 初次配置 | [SETUP.md](workspace/SETUP.md) |
| 职位评估、简历、求职信和面试 | [job-application-assistant](workspace/.claude/skills/job-application-assistant/SKILL.md) |
| 搜索职位 | [job-scraper](workspace/.claude/skills/job-scraper/SKILL.md) |
| 能力补齐 | [upskill](workspace/.claude/skills/upskill/SKILL.md) |
| 具体命令和评审规则 | [命令目录](workspace/.claude/commands/)及[Agent 目录](workspace/.claude/agents/) |

上游 `.claude/` 是方法来源，其他宿主可以按需读取对应文件，将工具名映射为实际可用能力。岗位搜索门户主要面向丹麦市场，其他地区需要选择合适的门户；不能把未运行的门户查询或文档编译说成成功。候选人材料的对外提交、发信和公开发布须有用户明确授权。

## 来源

上游仓库：[MadsLorentzen/ai-job-search](https://github.com/MadsLorentzen/ai-job-search)，来源目录为仓库根目录。`workspace/` 保留上游模板与工具，未引入 Git 历史和 GitHub CI 配置；本文件为 Harn 新增的入口。许可见 [LICENSE](LICENSE)。
