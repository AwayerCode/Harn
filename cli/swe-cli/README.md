# SWE CLI 运行原型

本目录实现[系统设计](../../docs/system-design.md)的第一阶段：验证包含 Node 运行时的二进制、单次校验协议及外围 TS 子进程执行。尚未替换 SWE 的 Python 检查器，也尚未接入 harn 配套安装，不作为完整 SWE 发行包发布。

## 已实现能力

1. help/version 与 JSON stdin 请求，0/1/2 分别表示通过、违规、不可检查。
2. documents 检查使用构建时从唯一来源 `skills/swe/references/artifact-schemas/` 读取并嵌入的 schema，覆盖格式、局部架构模式与本地链接；规则摘要随结果返回，不另维护规则副本。
3. 显式配置的单文件 TS 扩展在子进程中执行；校验内容摘要、接口版本、超时与结果格式，扩展结果不能覆盖核心结果。
4. 读取的输入文件生成摘要，结束前复核内容；发现变化返回不可检查，不声称提供原子快照。
5. Node SEA 本机打包、macOS 签名与 SHA-256 输出。只验证了 macOS arm64，不代表六平台验收完成。

## 开发与打包

构建锁定 Node 22.22.3，开发依赖由 package-lock.json 固定。以下命令在本目录运行：

```bash
npm ci
npm run typecheck
npm test
npm run package
npm run test:binary
```

构建产物位于 `dist/`，不提交。package 仅打包当前机器平台；不伪装成交叉编译脚本。test:binary 将二进制单独复制到含空格和中文的临时目录，使用无 Node/Python 的 PATH，从不同工作目录运行核心和 TS 扩展；不是完整容器或三系统验证。

## 请求示例

这是原型接口，JSON 从 stdin 输入，不需要创建请求文件。projectRoot 必须使用真实的绝对项目路径；示例路径由使用者替换。

```json
{
  "protocolVersion": 1,
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

调用 `swe-cli check --request - --format json`，或将请求保存到授权文件后传入其路径。version 输出运行时版本和 standalone 标志。原型只支持 documents 检查，每次请求最多一个同名检查；其他 checkId 明确返回 unavailable。缺失文件和不支持的请求返回 2，不把环境缺口伪装成格式违规。整体 passed 不是任务完成结论。

## 外围 TS 契约

扩展是可信代码，不是安全沙箱。首版使用可擦除类型语法和 CommonJS 形式，不能使用静态 import、第三方包、相对模块导入或需转换的 TS 语法。提供给扩展的 require 仅允许 node:path、node:url、node:crypto、node:fs 和 node:fs/promises；这不是对恶意代码的权限隔离。

```typescript
module.exports = async function (context: { taskId: string }) {
  return {
    interfaceVersion: 1,
    status: 'passed',
    diagnostics: [],
    evidence: [context.taskId]
  };
};
```

在请求的 extensions 数组中显式配置 id、path、sha256、interfaceVersion、required、timeoutMs、input 和 readOnly。path 相对 projectRoot，真实路径不得越出项目；随包路径定位尚待分发层接入。sha256 必须对应文件内容；timeoutMs 为正整数，单位毫秒。输入和扩展总输出限制为 1 MiB，这是原型资源上限，不是性能承诺。

Review 只接受明确声明 readOnly 的已审查扩展，该标志本身不证明没有副作用。扩展返回 interfaceVersion、status、diagnostics、evidence，不接受额外字段。非通过结果必须包含诊断，诊断包含 ruleId、message 和可选 location。子进程异常、超时、无效输出、摘要或版本不匹配均为 unavailable。

核心报告由父进程保存。必需扩展无法检查时整体退出 2，违规退出 1；可选扩展的问题作为显式 warnings 保留。扩展不接收可写的核心结果，不能通过返回 source 或 summary 改写核心结论。

## 后续迁移边界

设计、产物、范围和系统结构检查尚未迁移；尚未实现配套安装清单、读写锁、恢复事务、项目配置和 Agent 接入。完整协议还需机器 schema 和迁移对照验证。当前 schema 由构建内嵌，发行资源版本绑定仍待实现，不声称已验证外置资源完整性。

现有 SWE 工作流继续调用 Python，避免原型被误认为完整替代。后续需完成六个原脚本的语义对照，接入共享执行规范的循环，然后接入 harn 的平台配套安装与恢复，最后进行六平台发行验收。
