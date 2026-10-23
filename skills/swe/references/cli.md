# 检查工具

检查由配套 Go 二进制执行；Skill 保存方法、规范、schema 与模板。`runtime.json` 绑定工具版本及六个平台资产的 SHA256。使用支持配套安装的 harn 安装或更新本 Skill，二进制位于 `<SWE目录>/bin/swe-cli`；Windows 使用 `swe-cli.exe`。缺少工具或版本与清单不一致时先补装，不回退其他检查器。

## 调用

命令可从任意目录执行，显式传入项目根目录。文档路径可相对于项目根目录；项目外文件与符号链接越界返回不可检查。

```bash
<SWE目录>/bin/swe-cli version
<SWE目录>/bin/swe-cli documents --root <项目根目录> --kind system-design --file <文档路径>
<SWE目录>/bin/swe-cli documents --root <项目根目录> --kind system-design --profile architecture --file <文档路径>
<SWE目录>/bin/swe-cli design --root <项目根目录> --system-design <路径> --interface-protocol <路径>
<SWE目录>/bin/swe-cli artifacts --root <项目根目录> --kind ui --file <交互规范路径> --protocol <接口协议路径>
<SWE目录>/bin/swe-cli artifacts --root <项目根目录> --kind test --file <测试设计路径>
<SWE目录>/bin/swe-cli structure --root <项目根目录> --design <系统设计路径> --scope <相对路径>
```

纯界面动作的 artifacts 检查可以省略 protocol；业务操作需要协议依据。structure 只核对显式文件位置、入口与映射，不解析源码依赖或导出语义。输出的 unverified 项仍须人工审查。

## 范围与只读核对

```bash
<SWE目录>/bin/swe-cli scope snapshot --root <Git根目录> --baseline <项目外全新文件> --allow-file <相对路径>
<SWE目录>/bin/swe-cli scope check --root <Git根目录> --baseline <原文件>
<SWE目录>/bin/swe-cli scope readonly --root <Git根目录>
<SWE目录>/bin/swe-cli scope readonly --root <Git根目录> --expected <原sha256字段完整值>
```

可重复 allow-file 或 allow-dir。snapshot 只创建全新的项目外基线，check 不接受新范围。readonly 不写文件；sha256 字段包含格式前缀，调用者保存并原样传回。旧基线与旧指纹不兼容，进行中的任务继续使用原工具；不得通过重新建立基线掩盖修改。

范围检查需要 Git，单次 Git 命令最多 10 秒。跟踪与非忽略文件、显式可写范围内的忽略文件、子模块引用与内容均纳入相应检查；不自动初始化子模块，不允许父仓写入范围覆盖子模块。

## 结果与边界

普通检查输出统一 JSON，退出码 0 为必需检查通过、1 为违规、2 为不可检查。缺失工具、无效请求、读取失败、越界或执行中观察到输入变化均不能当作通过。

机器调用可继续使用 `swe-cli check --request <JSON文件或-> --format json`，协议版本 2；同一请求不能重复 checkId，多份同类型文档分别调用。范围操作使用独立 scope 命令，因为 snapshot 有显式写入副作用。文件观察不是原子快照，检查通过不替代业务、架构、测试或内容语义审查。

## YAML 模块架构

`architecture validate/check/render` 分别提供设计校验、文件结构与 Go 模块依赖检查、只读 HTML 生成。字段、采用规则、局部设计和执行示例见[模块架构规范](architecture/guide.md)。
