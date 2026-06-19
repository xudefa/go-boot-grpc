# 贡献指南

感谢你对 go-boot 项目的关注！本文档将帮助你在 10 分钟内完成开发环境搭建并提交第一个 Pull Request。

## 前提条件

- Go 1.21 或更高版本
- Git
- 熟悉的代码编辑器（推荐 VS Code 或 GoLand）

## 快速开始

### 1. 克隆仓库

```bash
git clone https://github.com/xudefa/go-boot-gin.git
cd go-boot-gin
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并生成覆盖率报告
go test -cover ./...

# 运行测试并检测数据竞争
go test -race ./...
```

### 4. 代码格式化

```bash
# 格式化代码
go fmt ./...

# 运行 lint 检查（需安装 golangci-lint）
golangci-lint run
```

## 开发流程

### 分支策略

- `master` — 主分支，保持稳定
- `feature/*` — 功能开发分支
- `fix/*` — 修复分支
- `docs/*` — 文档更新分支

### 提交规范

提交信息应遵循以下格式：

```
<type>: <description>

[optional body]
```

常用 type：

- `feat` — 新功能
- `fix` — 修复 bug
- `docs` — 文档更新
- `refactor` — 代码重构
- `test` — 测试相关
- `chore` — 构建/工具相关

示例：

```
feat(core): implement container interface with singleton/prototype scope

fix(aop): resolve pointcut matching issue for interface methods

docs(readme): add installation instructions and usage examples
```

### 提交 Pull Request

1. Fork 本仓库
2. 创建功能分支：`git checkout -b feature/my-feature`
3. 提交更改：`git commit -m 'feat: add my feature'`
4. 推送分支：`git push origin feature/my-feature`
5. 在 GitHub 上创建 Pull Request

### PR 要求

- [ ] 代码通过所有测试（`go test ./...`）
- [ ] 代码已格式化（`go fmt ./...`）
- [ ] 新增功能包含相应的测试
- [ ] 更新相关文档（如适用）
- [ ] 提交信息遵循规范

## 代码规范

### 命名规范

- 包名：小写，避免下划线（如 `container` 而非 `ioc_container`）
- 导出标识符：大驼峰（如 `Container`、`Register`）
- 非导出标识符：小驼峰（如 `container`、`register`）
- 错误变量：以 `Err` 前缀（如 `ErrNotFound`）
- 接口：以 `er` 结尾（如 `Reader`、`Writer`）

### 注释规范

- 使用中文注释，保持国际化友好
- 导出函数/类型必须有 godoc 注释
- 注释应说明"为什么"而非"做什么"

### 错误处理

- 不忽略任何错误
- 使用 `%w` 包装错误以保留错误链
- 使用哨兵错误表示框架级错误

```go
var ErrNotFound = errors.New("not found")

if err := find(id); err != nil {
    return fmt.Errorf("lookup failed: %w", err)
}
```

## 测试要求

### 测试命名

```
TestFunctionName_Condition_ExpectedBehavior
```

示例：

```go
func TestContainer_RegisterAndGet_Success(t *testing.T)
func TestContainer_GetT_NotFound_ReturnsError(t *testing.T)
```

### 覆盖率目标

- 核心模块：80%+
- 集成模块：70%+

### 运行特定测试

```bash
# 运行特定包的测试
go test ./core/container/... -v

# 运行特定测试函数
go test ./core/container/... -run TestContainer_Register -v

# 生成覆盖率报告
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 架构设计原则

### 零外部依赖

核心框架不引入任何外部依赖，仅使用 Go 标准库。集成模块通过独立仓库提供第三方库支持。

### 接口优先

优先定义接口，再提供默认实现。这使得用户可以轻松替换实现。

### 函数式选项

使用函数式选项模式提供灵活的配置：

```go
func NewContainer(opts ...ContainerOption) Container {
    cfg := defaultContainerConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    // ...
}
```

## 问题反馈

- **Bug 报告**：创建 Issue 并添加 `bug` 标签
- **功能请求**：创建 Issue 并添加 `enhancement` 标签
- **问题咨询**：创建 Issue 并添加 `question` 标签

## 行为准则

- 尊重所有贡献者
- 接受建设性批评
- 关注问题而非个人
- 欢迎新贡献者

感谢你的贡献！