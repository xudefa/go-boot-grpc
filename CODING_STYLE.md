# go-boot 项目代码规范

## 1. 总体原则

### 1.1 设计哲学
- **清晰优于巧妙**：代码应该易于理解和维护
- **简单优于复杂**：优先选择简单直接的实现方式
- **可读性第一**：代码首先是给人阅读的，其次才是给机器执行的
- **零外部依赖**：核心框架不引入外部依赖，仅使用Go标准库

### 1.2 代码质量要求
- 所有代码必须通过静态检查（golangci-lint）
- 重要功能必须有单元测试覆盖
- 关键逻辑需要有充分的注释说明

## 2. 命名规范

### 2.1 包命名
- 包名全部小写
- 多个单词用连字符连接（如 `user-service`）
- 除 `main` 包外，其他包名应与最内层目录名保持一致
- 核心包名采用简洁语义化命名（如 `core`, `aop`, `boot`）

### 2.2 标识符命名
- **导出标识符**：大写驼峰（`UserID`, `GetUser`）
- **非导出标识符**：小写驼峰（`userID`, `getUser`）
- **常量**：使用驼峰命名（`MaxConnections`, `DefaultTimeout`），避免使用全大写下划线
- **测试函数**：`TestFunctionName_Condition_ExpectedBehavior`
- **错误变量**：以 `Err` 前缀（`ErrNotFound`, `ErrInvalidInput`）
- **接口命名**：
  - 简单接口以 `er` 后缀（`Reader`, `Writer`）
  - 功能性接口以功能描述命名（`Logger`, `Cache`, `Repository`）

### 2.3 文件命名
- Go源文件使用小写蛇形命名（`container.go`, `bean_factory.go`）
- 测试文件以 `_test.go` 结尾
- 配置文件使用 `.yaml` 或 `.json` 扩展名

## 3. 代码结构与组织

### 3.1 目录结构
```
go-boot/
├── core/           # IoC容器核心
├── aop/            # AOP框架
├── boot/           # 应用启动器
├── context/        # 应用上下文
├── environment/    # 环境配置
├── condition/      # 条件判断
├── event/          # 事件系统
├── data/           # 数据访问抽象
├── cache/          # 缓存抽象
├── config/         # 配置管理
├── log/            # 日志抽象
├── net/            # 网络接口抽象
├── health/         # 健康检查
├── metrics/        # 指标收集
├── tracing/        # 分布式追踪
├── actuator/       # 运维端点
└── schedule/       # 定时任务
```

### 3.2 包内文件组织
- `package` 声明后是包级别的文档注释
- 常量定义
- 类型定义（struct, interface, type alias）
- 变量声明
- 公共函数
- 私有函数
- 方法定义（按接收者分组）

### 3.3 导入规范
```go
import (
    // 标准库
    "context"
    "fmt"
    "sync"

    // 项目内部包
    "github.com/xudefa/go-boot/core"
    "github.com/xudefa/go-boot/log"
)
```

## 4. 注释与文档

### 4.1 注释语言
- 使用中文注释，保持国际化友好
- 重点注释应说明"为什么这样做"而不是"做了什么"

### 4.2 包注释
```go
// Package core 提供了一个轻量级的依赖注入(DI)容器实现,灵感来自Spring Framework的IoC容器.
//
// # 核心功能
//
//   - Bean注册: 支持通过实例、工厂函数或类型注册bean
//   - 依赖注入: 支持字段注入(通过inject标签)和构造函数注入
//   - 作用域管理: 支持单例(Singleton)和原型(Prototype)作用域
package core
```

### 4.3 类型和函数注释
```go
// CalculateDiscount 计算应用分级折扣后的最终价格。
// 折扣根据订单数量逐步应用：每个等级解锁额外的百分比减免。
// 如果数量无效或基础价格在应用折扣后会导致负值，则返回错误。
//
// 参数:
//   - basePrice: 任何折扣前的原始价格（必须为非负数）
//   - quantity: 订单的数量（必须为正数）
//   - tiers: 按最小数量阈值排序的折扣等级切片
//
// 返回最终折扣价格，四舍五入到小数点后两位。
// 如果 basePrice 为负数，返回 ErrInvalidPrice。
// 如果 quantity 为零或负数，返回 ErrInvalidQuantity。
func CalculateDiscount(basePrice float64, quantity int, tiers []DiscountTier) (float64, error) {
    // implementation
}
```

### 4.4 行内注释
- 对于复杂的逻辑，使用行内注释解释原因
- 使用 TODO、FIXME、NOTE 等标记特殊注释
- 重要决策的权衡考虑应有注释说明

## 5. 代码风格

### 5.1 行长度
- 无严格的行长度限制，但超过 ~120 字符时应考虑换行
- 函数调用超过 4 个参数时，每个参数独占一行

```go
// Good
result := ProcessData(
    input,
    config,
    logger,
    validator,
)

// Avoid
result := ProcessData(input, config, logger, validator, processor, transformer)
```

### 5.2 变量声明
- 非零值使用短变量声明 `:=`
- 零值初始化使用 `var`
- 切片和映射必须初始化，不允许为 nil

```go
var count int              // 零值，使用 var
name := "default"          // 非零值，使用 :=
users := []User{}          // 初始化为空切片，非 nil
m := map[string]int{}      // 初始化为空映射，非 nil
```

### 5.3 控制流
- 优先处理错误和边界条件（早期返回）
- 消除不必要的 `else`
- 复杂条件提取为命名布尔变量

```go
// Good - 早期返回
func process(data []byte) (*Result, error) {
    if len(data) == 0 {
        return nil, errors.New("empty data")
    }
    
    if !isValid(data) {
        return nil, errors.New("invalid data format")
    }
    
    parsed, err := parse(data)
    if err != nil {
        return nil, fmt.Errorf("parse failed: %w", err)
    }
    
    return transform(parsed), nil
}

// Good - 命名布尔变量
isAdmin := user.Role == RoleAdmin
isOwner := resource.OwnerID == user.ID
isPublicVerified := resource.IsPublic && user.IsVerified
if isAdmin || isOwner || isPublicVerified {
    allowAccess()
}
```

### 5.4 函数设计
- 函数应简短专注，单一职责
- 参数不超过 4 个，超过时使用选项结构体
- `context.Context` 总是第一个参数
- 使用 `range` 迭代优于索引循环

## 6. 错误处理

### 6.1 错误处理原则
- 不忽略任何返回的错误
- 使用 `fmt.Errorf` 和 `%w` 包装错误
- 提供清晰的错误信息
- 使用哨兵错误（sentinel errors）表示框架错误

```go
var (
    ErrNotFound = errors.New("item not found")
    ErrInvalidInput = errors.New("invalid input provided")
)

// Good - 包装错误
if err := validate(input); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

### 6.2 自定义错误类型
```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}
```

## 7. 泛型使用

### 7.1 泛型最佳实践
- 优先使用泛型实现类型安全的API
- 泛型约束使用具体类型而非过于宽泛的约束
- 避免过度泛型化，只在确实需要类型安全时使用

```go
// Good - 类型安全的Repository
type Repository[T any] interface {
    Save(entity T) error
    FindByID(id string) (T, error)
    FindAll() ([]T, error)
}

// Good - 泛型工具函数
func ZeroOf[T any]() T {
    var zero T
    return zero
}
```

## 8. 特定领域规范

### 8.1 IoC 容器规范
- 使用 `core.New()` 创建容器
- 启用字段注入：`core.EnableFieldTag(true)`
- Bean 注册使用 `container.Register("id", core.Bean(value))`
- 字段注入使用 `inject:"beanId"` 结构体标签
- 工厂函数使用 `core.Factory(func(c core.Container) (any, error))`

### 8.2 AOP 规范
- 通知类型：`aop.Before`, `aop.After`, `aop.Around`, `aop.AfterReturning`, `aop.AfterThrowing`
- 切点匹配器：`aop.MatchByName`, `aop.MatchByPrefix`, `aop.MatchByRegex` 等
- 通过 `aop.WithOrder(n)` 控制通知执行顺序
- Around 通知必须调用 `proceed` 使调用链继续

### 8.3 函数式选项模式
```go
// Good - 函数式选项
container.Register("service",
    core.Bean(&Service{}),
    core.Singleton(),
    core.DependsOn("db"),
    core.Init(func(s *Service) error { return s.Start() }),
    core.Condition(func(c core.Container) bool { return c.Has("db") }),
)
```

## 9. 测试规范

### 9.1 测试命名
- 表格驱动测试：`TestFunctionName_TableDriven`
- 边界条件测试：`TestFunctionName_BoundaryCondition_ExpectedBehavior`
- 集成测试：`TestIntegration_Scenario_ExpectedBehavior`

### 9.2 测试结构
```go
func TestCalculateDiscount(t *testing.T) {
    tests := []struct {
        name          string
        basePrice     float64
        quantity      int
        tiers         []DiscountTier
        expected      float64
        expectError   bool
    }{
        {
            name:      "normal calculation",
            basePrice: 100.0,
            quantity:  10,
            expected:  95.0, // 5% discount
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := CalculateDiscount(tt.basePrice, tt.quantity, tt.tiers)
            
            if tt.expectError {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## 10. 代码审查清单

### 10.1 功能正确性
- [ ] 逻辑正确，边界条件处理得当
- [ ] 错误处理完善，没有忽略错误
- [ ] 并发安全，正确使用同步原语

### 10.2 代码质量
- [ ] 代码清晰易懂，符合命名规范
- [ ] 无冗余代码，遵循 DRY 原则
- [ ] 注释恰当，解释复杂逻辑

### 10.3 性能考虑
- [ ] 无明显性能瓶颈
- [ ] 内存分配合理
- [ ] 循环和递归使用得当

### 10.4 安全性
- [ ] 输入验证充分
- [ ] 无安全漏洞风险
- [ ] 敏感信息妥善处理