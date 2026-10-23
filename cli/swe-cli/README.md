# SWE CLI

[系统设计](../../docs/system-design.md)中的 Go 单次校验核心。Skill 保存方法与规范，CLI 执行确定性检查；业务测试和构建仍由 Agent 按项目约定调用。

## 检查能力

| 入口 | 覆盖 |
| --- | --- |
| documents | 五类文档格式、表格、枚举、本地链接与 architecture 局部模式 |
| design | 模块依赖、协议索引与定义、所有权、代码映射及双向文档引用 |
| artifacts | UI 与协议、风险与场景的显式对应关系 |
| structure | 设计声明的代码位置、入口文件和范围映射 |
| scope snapshot/check/readonly | 可写范围基线、工作区变更和只读指纹，含子模块 |

检查不判断任务整体完成，不解析源码依赖，不替代人工语义审查。结果保留已检查证据和未覆盖项。只有 scope snapshot 创建全新的项目外基线；其他检查不写项目。

## YAML 模块架构

新增 `architecture validate/check/render` 入口，执行结构化设计校验、目录归属与 Go 跨模块导入检查，并生成可展开、可追踪依赖的只读 HTML。采用规则、限制与示例见 [YAML 模块架构](../../skills/swe/references/architecture/guide.md)。`validate --profile architecture` 用于层设计交接，完整验收要求存在模块；该入口不并入 JSON 协议版本 2。

## 使用

```bash
swe-cli documents --root /absolute/project --kind system-design --file docs/system-design.md
swe-cli design --root /absolute/project --system-design docs/system-design.md --interface-protocol docs/interface-protocol.md
swe-cli artifacts --root /absolute/project --kind test --file docs/test-design.md
swe-cli structure --root /absolute/project --design docs/system-design.md --scope src
swe-cli scope snapshot --root /absolute/project --baseline /tmp/new-baseline.json --allow-dir src
swe-cli scope check --root /absolute/project --baseline /tmp/new-baseline.json
swe-cli scope readonly --root /absolute/project
```

完整参数与边界见 [SWE CLI 入口](../../skills/swe/references/cli.md)。readonly 的 sha256 字段含格式前缀，保存后原样传给 expected。旧脚本基线和指纹明确拒绝，不自动重建或迁移正在使用的基线。


## JSON 协议

普通检查支持 `swe-cli check --request <file|-> --format json`。协议版本 2，旧扩展字段拒绝。退出码 0 为全部必需检查通过，1 为违规，2 为不可检查；不可检查优先，所有诊断仍保留。至少包含一个必需检查，同一请求不能重复 checkId。

```json
{
  "protocolVersion": 2,
  "taskId": "document-check",
  "round": 1,
  "mode": "review",
  "projectRoot": "/absolute/project",
  "checks": [
    {
      "checkId": "design",
      "required": true,
      "input": {
        "systemDesign": "docs/system-design.md",
        "protocol": "docs/interface-protocol.md"
      }
    }
  ]
}
```

各 input 字段：documents 使用 kind/file/profile；design 使用 systemDesign/protocol；artifacts 使用 kind/file/protocol；structure 使用 design/scopes。范围操作独立于 check，因为 snapshot 有明确写入副作用。

结果包含 runId、核心版本、规则与输入摘要，并复核观察到的文件变化。不是文件系统原子快照。scope 另使用版本化 Git 工作区清单与摘要。

## 开发与构建

需要 Go 1.24+；Git 用于范围检查。运行普通文档关系检查不需要 Git。二进制不依赖 Python、Node 或 Go 运行环境，YAML 与 JSON Schema 库编入二进制，无额外运行时依赖。

```bash
go generate ./...
go test -race ./...
go vet ./...
scripts/build-release.sh 0.4.0-alpha.1
go run ./internal/genbinding --version 0.4.0-alpha.1 --dist dist --root ../..
```

schema 从唯一来源 `skills/swe/references/artifact-schemas/` 生成并编入二进制；测试检查来源一致性、模块契约引用和 Skill 链接。生成文件不可手改。

迁移用例保存在 internal/app/testdata，包含由旧 Python 检查器生成的 95 个正反例及预期退出码，执行不需要 Python。另有显式场景引用、权限、符号链接、子模块、基线保护、工具版本与摘要回归测试。独立二进制测试使用无语言运行时 PATH。

## 配套分发

构建输出 macOS、Linux、Windows 的 amd64/arm64 二进制及 SHA256SUMS。genbinding 将确切发行版本与六平台摘要写入 SWE 的 runtime.json。上传发行物时必须使用生成清单所对应的原始二进制，重新构建后需重新生成清单。

支持配套清单的 harn 将工具暂存于所选 Skill 的 bin 目录，摘要通过后与资源一起提交；缺平台、下载或摘要失败不启用新资源。更新保护本地修改，status 检查二进制内容。CLI 在 bin 目录运行时核对旁边清单的版本与自身摘要。

当前清单绑定 0.4.0-alpha.1，尚需单独发布对应资产；旧 0.2.0-alpha.1 不包含这些检查。发布前可在本地使用构建产物验收，不应将未发布清单推送给业务项目使用。崩溃恢复、配套修复命令、安装读锁和最低系统版本验收仍待完成。六平台构建不等于跨平台运行验证，当前实际运行覆盖 macOS arm64。
