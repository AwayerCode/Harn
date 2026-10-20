# SWE CLI

[系统设计](../../docs/system-design.md)中的 Go 单次校验核心。现有文档检查已从 Node 原型迁移，Node 打包、TS 扩展宿主和扩展协议已移除。方法与项目适配放在 Skill 中，Agent 按 Skill 调用项目测试和构建。

## 当前能力

- documents 校验文档格式、局部 architecture 模式与本地链接，不判断语义正确性或任务是否完成。
- JSON 请求与结果，退出码 0 表示通过、1 表示违规、2 表示不可检查。必需检查不可用优先于违规；可选检查问题保留在 warnings。
- schema 从唯一来源 `skills/swe/references/artifact-schemas/` 生成并编入二进制。生成文件不可手改，测试检查其与来源一致。
- 结果包含 runId、核心版本、规则与输入摘要；结束前复核已观察文件的内容、路径解析和链接存在性。不是文件系统原子快照。
- 检查不写项目文件，不加载外部脚本，不依赖系统 Node、TS、Python 或 Go。

## 开发与构建

使用 Go 1.24 或更高版本，在本目录执行：

```bash
go generate ./...
go test -race ./...
go vet ./...
scripts/build-release.sh 0.2.0-dev
```

无第三方 Go 依赖。生成与构建需要完整源码仓；运行二进制不需要源码。迁移对照测试在存在 Python 时运行原检查器，缺少 Python 会明确跳过该组测试；正式校验不会回退 Python。

构建输出在忽略的 dist 目录，包括 macOS、Linux、Windows 的 amd64/arm64 二进制及 SHA256SUMS。独立运行测试将二进制置于含中文和空格的临时目录，在没有语言运行时的 PATH 下，从不同工作目录执行 Review 并检查无项目写入。六平台交叉构建不替代各系统实际运行验收；当前只在 macOS arm64 实测。

## 请求与结果

原型协议升级到 2，旧版本及 extensions 字段明确拒绝，不静默忽略旧扩展。projectRoot 使用真实的绝对项目路径；文件路径可相对于它。

```json
{
  "protocolVersion": 2,
  "taskId": "document-check",
  "round": 1,
  "mode": "design",
  "projectRoot": "/absolute/project",
  "checks": [
    {
      "checkId": "documents",
      "required": true,
      "input": {
        "kind": "business-model",
        "file": "docs/business-model.md"
      }
    }
  ]
}
```

调用 `swe-cli check --request - --format json` 从 stdin 读取，或将短横线替换为请求文件路径。请求上限 1 MiB，至少包含一个必需检查，不允许重复 checkId。version 输出核心、协议和 Go 运行时版本。

支持 business-model、system-design、interface-protocol、interaction-standards、test-design；system-design 可指定 profile 为 architecture。缺失文件、无效 UTF-8、越出项目的真实路径、未知检查或无效请求返回不可检查。只支持每次一个 documents 检查，多文档需分别调用；未实现的检查不会报告通过。

## 后续边界

其余 Python 检查、范围基线、资源绑定、安装读写锁、恢复事务、项目配置及 harn 配套安装仍待实现。现有 SWE 工作流继续使用原 Python 检查器，待完整迁移与验证后接入 Go CLI。当前版本不是完整 SWE 发行包；核心通过不能替代业务验收或 Agent 的完成判断。
