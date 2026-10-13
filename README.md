# Skills

个人 Skill 的独立源码仓，保存可复用的专业方法、规则、资源、脚本与检查标准。Skill 源码在本仓维护，使用项目通过安装器按发布版本调用，不在使用项目维护另一份源码。

## 入口

Skill 列表见 [Skills 总纲](0-Skills.md)。各 Skill 以自己的 `SKILL.md` 为入口，细节、模板、脚本和测试保留在所属 Skill 中。

## 安装与分发

对已发布版本，在目标项目执行 `npx @awayercode/skills`，安装到 `.agents/skills/`。安装后的目录是使用快照，修改与发布回到本仓。

发布工具位于 `publish/`，包名继续使用 `@awayercode/skills`。本地检查执行 `npm test --prefix publish`，构建包执行 `node publish/build.mjs`。更新尚未发布时，可用构建快照在目标项目验证，不将本地验证等同于已发布版本。

## 源码来源

初始内容迁自 Sisi 的 `Skills/`，原有版本历史保留在 [Sisi 源码历史](https://github.com/AwayerCode/Sisi/tree/6c8f343508b5791075312bf789a0ee11fefbd261/Skills)。后续以本仓为唯一源码位置。
