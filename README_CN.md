# OpenMigrate

[![CI](https://github.com/juan-xin-cai/openmigrate/actions/workflows/ci.yml/badge.svg)](https://github.com/juan-xin-cai/openmigrate/actions/workflows/ci.yml)
[![Release Please](https://github.com/juan-xin-cai/openmigrate/actions/workflows/release-please.yml/badge.svg)](https://github.com/juan-xin-cai/openmigrate/actions/workflows/release-please.yml)

把 Claude Code 环境迁到另一台 Mac，不必整份家目录一起搬。

OpenMigrate 会把一部分 Claude Code 数据，连同需要保留的 Claude Desktop 配置，打成加密的 `.ommigrate` 包；导入前可先查看元信息、检查环境、改写机器相关路径，导入时会做冲突检查，必要时还能一键回滚。

[English](./README.md)

## 这是什么

现在这版主要面向 macOS 上的 Claude Code `v2`，并兼顾 Claude Desktop `v1` 里跟迁移有关的数据。

适合这些场景：

- 换电脑，想把 Claude Code 的配置和项目资料带过去
- 不想直接拷整份 `~/.claude`
- 想先看包里有什么，再决定导不导
- 导入前想先做账号校验、冲突检查
- 出问题时，想能退回导入前状态

## 目前覆盖哪些内容

会带走的内容：

- Claude Code 的设置、项目历史、项目数据、skills、plugins、agents、commands
- Claude Desktop 的部分配置与 MCP 相关文件
- 可以安全迁移的 Desktop session 数据
- 需要按目标机器改写的项目路径、家目录路径

不会带走，或会先清洗的内容：

- `.claude/sessions/**` 原始 session 记录
- Claude Desktop 的 `.audit-key`
- 支持范围内 JSON 里的 OAuth、token 一类字段
- 不该原样复制的设备相关值
- Claude Desktop 的运行时 bundle、VM 目录

## 命令一览

| 命令 | 用途 |
| --- | --- |
| `openmigrate doctor [package-or-meta]` | 导出前或导入前先检查环境 |
| `openmigrate export` | 导出加密迁移包 |
| `openmigrate inspect <package.ommigrate>` | 看包的元信息 |
| `openmigrate import <package.ommigrate>` | 导入包，带路径映射和冲突处理 |
| `openmigrate rollback --snapshot latest` | 回滚最近一次导入 |

常用参数：

- `export --only settings,projects,skills`
- `export --exclude sessions`
- `export --no-history`
- `import --yes`
- `import --skip-desktop-session-check`
- 任意命令都可加 `--verbose`

## 安装

### 从源码编译

```bash
git clone https://github.com/juan-xin-cai/openmigrate.git
cd openmigrate
CGO_ENABLED=0 go build -o openmigrate ./cmd/openmigrate
```

要求：

- macOS
- Go 1.16+
- 已安装 Claude Code，且命令名为 `claude`

### 发行包

仓库已经带上 CI 和 release 流程。等打出正式 tag 后，可直接从 [GitHub Releases](https://github.com/juan-xin-cai/openmigrate/releases) 下载 macOS 通用二进制包和校验文件。

## 快速上手

### 1. 在源机器导出

```bash
./openmigrate export --out ~/Desktop/openmigrate
```

命令会提示输入密码，随后生成：

- `*.ommigrate` 加密包
- `*.meta.json` 元信息文件

如果要走非交互流程，可以先设环境变量：

```bash
export OPENMIGRATE_PASSPHRASE='your-passphrase'
./openmigrate export --out ~/Desktop/openmigrate
```

### 2. 先看包内容

```bash
./openmigrate inspect ~/Desktop/openmigrate/openmigrate.ommigrate
```

### 3. 在目标机器做检查

```bash
./openmigrate doctor ~/Desktop/openmigrate/openmigrate.ommigrate
```

如果包里带了 Claude Desktop 数据，先把 Full Disk Access、账号不一致这些问题处理掉，再导入。

### 4. 导入

```bash
./openmigrate import ~/Desktop/openmigrate/openmigrate.ommigrate
```

默认会做这些事：

- 给出目标家目录和项目路径的建议映射
- 先做冲突检查，交互式选择怎么处理
- Desktop session 存在时，校验 Claude Desktop 账号
- 写入前先做快照，方便回滚

### 5. 需要时回滚

```bash
./openmigrate rollback --snapshot latest
```

## 安全边界

- 导出包用密码加密
- 敏感字段在打包前就会清掉
- 导入不是直接全量覆盖，会先检查冲突
- Desktop session 账号不一致时，默认拦下
- 导入前会先做快照，方便退回

## 当前状态

项目刚开始公开，当前实现重点放在：

- macOS
- Claude Code `v2`
- 跟迁移连续性有关的 Claude Desktop `v1` 数据

Windows、Linux，以及更广的 agent 覆盖，还不在这版范围里。

## 开发

测试命令：

```bash
CGO_ENABLED=0 go test -count=1 ./...
```

仓库还带了这些东西：

- `main` 分支上的 GitHub Actions CI
- `release-please` 发布流程
- macOS 打包脚本 [`scripts/release-macos.sh`](./scripts/release-macos.sh)

## 参与贡献

欢迎提 issue 和 PR，尤其是这些方向：

- 迁移安全
- 路径改写覆盖面
- Claude Desktop 兼容性检查
- 打包和发布体验

如果要改包格式，或要扩大支持范围，最好先开个 issue，把迁移保证说清楚。
