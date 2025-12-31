# Actime 开发规范

## 代码风格

### Go代码规范
遵循 [Effective Go](https://go.dev/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### 命名规范

#### 包命名
- 使用小写单词
- 简短、有意义
- 避免下划线或驼峰
- 示例: `config`, `logger`, `tracker`

#### 变量命名
- 使用驼峰命名法
- 导出变量首字母大写
- 私有变量首字母小写
- 示例: `appName`, `StartTime`, `isActive`

#### 常量命名
- 使用驼峰命名法
- 导出常量首字母大写
- 示例: `MaxRetries`, `DefaultTimeout`

#### 函数命名
- 使用驼峰命名法
- 导出函数首字母大写
- 私有函数首字母小写
- 示例: `GetActiveWindow()`, `checkIdleTime()`

#### 接口命名
- 使用驼峰命名法
- 单方法接口以方法名+er结尾
- 示例: `type Detector interface { Detect() }`

### 文件组织

#### 单一职责
- 每个文件只负责一个主要功能
- 文件名应反映其内容
- 示例: `config.go`, `tracker.go`

#### 导入顺序
```go
import (
    // 标准库
    "context"
    "log"
    "time"

    // 第三方库
    "github.com/spf13/cobra"

    // 项目内部包
    "github.com/weii/actime/internal/config"
)
```

### 代码格式
- 使用 `gofmt` 格式化代码
- 使用 `goimports` 管理导入
- 行长度建议不超过120字符

## 注释规范

### 包注释
每个包都应有注释，说明包的用途

```go
// Package config 提供配置管理功能
package config
```

### 导出函数注释
所有导出函数都应有注释

```go
// LoadConfig 从指定路径加载配置文件
// 如果文件不存在，返回默认配置
func LoadConfig(path string) (*Config, error) {
    // ...
}
```

### 复杂逻辑注释
复杂算法或逻辑应添加注释

```go
// 使用滑动窗口检测活跃状态
// 窗口大小为5分钟，任何输入都会重置窗口
for {
    idleTime := getIdleTime()
    if idleTime < 5*time.Minute {
        // 用户活跃
        activeTime++
    }
}
```

### TODO注释
使用 `TODO:` 标记待完成的工作

```go
// TODO: 添加配置验证
func validateConfig(cfg *Config) error {
    // ...
}
```

## 错误处理

### 错误处理原则
- 不要忽略错误
- 尽早处理错误
- 提供有意义的错误信息

### 错误包装
使用 `fmt.Errorf` 包装错误

```go
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
```

### 错误检查
```go
file, err := os.Open(path)
if err != nil {
    return fmt.Errorf("failed to open file: %w", err)
}
defer file.Close()
```

## 日志规范

### 日志级别
- `Debug`: 调试信息
- `Info`: 一般信息
- `Warn`: 警告信息
- `Error`: 错误信息

### 日志格式
使用结构化日志

```go
logger.Info("Starting service",
    "version", version,
    "platform", platform,
)
```

### 错误日志
```go
logger.Error("Failed to connect to database",
    "error", err,
    "path", dbPath,
)
```

## 测试规范

### 测试文件
- 测试文件名: `xxx_test.go`
- 测试函数: `TestXxx`

### 测试示例
```go
func TestLoadConfig(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        want    *Config
        wantErr bool
    }{
        {
            name:    "valid config",
            path:    "testdata/config.yaml",
            want:    &Config{...},
            wantErr: false,
        },
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := LoadConfig(tt.path)
            if (err != nil) != tt.wantErr {
                t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("LoadConfig() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 测试覆盖率
- 核心逻辑覆盖率 >= 80%
- 工具函数覆盖率 >= 90%

## 提交规范

### 提交信息格式
```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type类型
- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建/工具链相关

### 示例
```
feat(tracker): implement window detection on Linux

Add X11-based window detection to get active window name.
This includes setup of X11 connection and window property reading.

Closes #1
```

## 分支管理

### 主分支
- `main`: 主分支，稳定版本
- `develop`: 开发分支

### 功能分支
- 命名: `feat/xxx`
- 从 `develop` 创建
- 合并回 `develop`

### 修复分支
- 命名: `fix/xxx`
- 从 `develop` 创建
- 合并回 `develop`

### 发布分支
- 命名: `release/v1.0.0`
- 从 `develop` 创建
- 合并回 `main` 和 `develop`

## 版本号规范

遵循 [Semantic Versioning](https://semver.org/)

- `MAJOR.MINOR.PATCH`
- `MAJOR`: 不兼容的API修改
- `MINOR`: 向下兼容的功能新增
- `PATCH`: 向下兼容的bug修复

## 性能规范

### 内存管理
- 使用 `sync.Pool` 复用对象
- 避免不必要的内存分配
- 及时释放大对象

### 并发控制
- 使用 `sync.Mutex` 保护共享资源
- 使用 `context.Context` 控制goroutine生命周期
- 避免goroutine泄漏

### 数据库操作
- 使用事务批量操作
- 及时释放数据库连接
- 合理使用索引

## 安全规范

### 输入验证
- 验证所有外部输入
- 清理用户输入
- 防止SQL注入

### 敏感信息
- 不记录敏感信息（密码、token等）
- 使用环境变量存储敏感配置
- 日志中脱敏敏感数据

### 权限控制
- 最小权限原则
- 验证文件权限
- 安全处理文件路径

## 文档规范

### 代码文档
- 所有导出类型、函数、常量都应有注释
- 注释应说明"是什么"和"为什么"
- 避免在注释中重复代码

### 项目文档
- README.md: 项目介绍和快速开始
- CONTRIBUTING.md: 贡献指南
- CHANGELOG.md: 变更日志

## 代码审查

### 审查要点
1. 代码风格是否符合规范
2. 错误处理是否完善
3. 日志是否合理
4. 测试是否充分
5. 性能是否达标
6. 安全是否有隐患

### 审查流程
1. 创建Pull Request
2. 自动化检查通过
3. 代码审查
4. 修改反馈
5. 合并

## 工具使用

### 必需工具
- `go`: Go编译器
- `gofmt`: 代码格式化
- `goimports`: 导入管理
- `golint`: 代码检查
- `go vet`: 静态分析

### 推荐工具
- `golangci-lint`: 综合代码检查
- `gocov`: 测试覆盖率
- `godoc`: 文档生成

### Makefile命令
```makefile
fmt:           # 格式化代码
lint:          # 代码检查
test:          # 运行测试
test-coverage: # 测试覆盖率
build:         # 构建项目
clean:         # 清理构建产物
```

## 常见问题

### Q: 如何处理平台特定代码?
A: 使用build tag区分平台
```go
//go:build linux

package platform

// Linux特定实现
```

### Q: 如何处理错误?
A: 尽早返回错误，包装错误信息
```go
if err != nil {
    return fmt.Errorf("failed to xxx: %w", err)
}
```

### Q: 如何保证代码质量?
A:
- 遵循代码规范
- 编写充分测试
- 进行代码审查
- 使用静态分析工具

## 参考资料

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Semantic Versioning](https://semver.org/)