# Skills 协作约定

本仓是个人 Skill 的独立源码位置。各 Skill 目录、配套引用、模板、脚本和测试都在这里维护。

## Skill 入口

各 Skill 以自己的 `SKILL.md` 为入口，执行时按需读取其规范、契约与脚本；配套资源、模板和测试保留在所属 Skill 中。

- [swe](swe/SKILL.md)：软件工程总入口，以共同标准、系统分层和模块契约组织专业设计、按层实现与集成验收；按任务加载相关依据和方法。
- [concise-response](concise-response/SKILL.md)：每轮对话使用，控制立场和表达；简洁不减少必须交付的内容。
- [knowledge-doc-guard](knowledge-doc-guard/SKILL.md)：用于知识文档读取、整理、写入和质量检查。

## 协作约定

先读取目标 Skill 的入口与相关资源，只修改当前任务涉及的内容。设计或修改 Skill 时保留有效方法、职责边界和检查要求，不因迁移重写专业知识。

变更脚本后，运行相应脚本测试；检查相关引用。部署路径和凭据不写入共享源码。

提交和推送遵循用户的具体授权，不因完成修改自动执行。

## 源码来源

初始内容迁自 Sisi 的 `Skills/`，原有版本历史保留在 [Sisi 源码历史](https://github.com/AwayerCode/Sisi/tree/6c8f343508b5791075312bf789a0ee11fefbd261/Skills)。后续以本仓为唯一源码位置。
