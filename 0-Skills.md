# Skills

本目录是个人 Skill 的统一源码位置。其他项目执行 `npx @awayercode/skills`，把发布版本写入 `.agents/skills/`，同名目录覆盖。执行时读取对应 `SKILL.md`，使用其自带的规范、契约与脚本。

## Skill 入口

- [design-skill](design-skill/SKILL.md)：按目标、规则、资源、路径设计、新建、优化和审查 Skill 及其内部模块。
- [swe](swe/SKILL.md)：软件工程总入口，按任务选择业务建模、系统与交互设计、代码与界面实现、测试或审查模块；模块关系、产物依赖及职责边界由该入口维护。
- [concise-response](concise-response/SKILL.md)：每轮对话使用，控制立场和表达；简洁不减少必须交付的内容。
- [knowledge-doc-guard](knowledge-doc-guard/SKILL.md)：用于知识文档读取、整理、写入和质量检查。
