# Skill 创作指南

这个仓库同时是 Skill 的源码仓库和分发源。新增一个 Skill 需要同时满足三件事：入口可被发现、结构可被加载、质量可被验证。

## 目录结构

每个 Skill 位于 `Skills/<name>/`，以 `SKILL.md` 为唯一入口：

```tree
Skills/<name>/
├── SKILL.md              # 入口：适用场景、执行主线、资料索引
├── agents/
│   └── openai.yaml       # 界面显示名、简述和默认提示词
├── references/           # 按需加载的规范、契约和方法
├── assets/               # 模板与可复制产物
└── scripts/
    └── tests/            # 校验脚本及其测试
```

新增 Skill 可复制 [`_template/`](_template/SKILL.md) 到 `Skills/<name>/`，替换入口和配套文件中的占位内容，再按需删减。

只创建当前 Skill 真正需要的目录。没有脚本就不建 `scripts/`，没有模板就不建 `assets/`。

## SKILL.md 要求

1. 文件以 YAML front matter 开头，只包含 `name` 和 `description`。
2. `name` 与目录名一致，全小写，使用连字符分隔。
3. `description` 说明“什么时候该用”，不罗列实现细节，不超过三句话。
4. 正文先写执行主线：入口先做哪些判断、按什么顺序推进、何时停止。
5. 资料用表格或链接指向 `references/`，不在入口重复完整方法。
6. 明确职责边界：本 Skill 负责什么、不负责什么、与相邻 Skill 如何衔接。

入口是索引不是全集。入口写判断与路由，方法、契约、模板放各自的相对文件。

## 资料分层

| 位置 | 内容 |
| --- | --- |
| `references/standards/` | 会被多个模块共用的规则和口径 |
| `references/*.json` | 机器可校验的输入输出契约 |
| `assets/` | 直接复制到目标项目的文档骨架 |
| `scripts/` | 可执行校验，必须带测试 |

同一规则只保留一处定义。模块文档引用共享规则，不复制第二份表述。

## 注册到索引

新增或改名 Skill 后，同步更新 `Skills/index.json`：

```json
{
  "name": "example",
  "path": "Skills/example",
  "summary": "一句话说明它解决什么问题",
  "tags": ["tag-a", "tag-b"]
}
```

索引由 `harn search` 读取，`name` 必须唯一，`path` 必须在 `Skills/` 下。索引校验按下节的统一验证约定执行。

## 校验

按仓库 [AGENTS.md 的本地验证约定](../AGENTS.md#本地验证)，执行当前改动涉及的 Skill、索引与文档检查。

新增脚本必须附带测试；测试放在 `scripts/tests/`，使用 `unittest`，不依赖网络和固定绝对路径。

## 命名与边界

- 一个 Skill 解决一类任务，不按阶段或文件类型拆分。
- 名称表达任务对象，不用内部代号。
- 与已有 Skill 重叠时优先扩展已有入口，不新建平行规则。
- 不把项目专属路径、凭据或部署信息写进 Skill。
