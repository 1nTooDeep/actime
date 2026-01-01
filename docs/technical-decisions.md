# Actime 技术决策文档

## 项目概述
Actime 是一个跨平台（Windows + Linux）的后台进程，用于统计用户在前台软件上的真实活跃时间。

## 技术栈选型

### 核心原则
- 高性能、低内存占用
- 优先使用标准库
- 最小化第三方依赖

### 依赖库清单

| 库名 | 版本 | 用途 | 必需性 |
|------|------|------|--------|
| `modernc.org/sqlite` | v1.28.0 | 数据库（纯Go） | 必需 |
| `github.com/kardianos/service` | v1.2.2 | 跨平台服务管理 | 必需 |
| `github.com/BurntSushi/xgb` | latest | Linux X11协议 | 必需（Linux） |
| `github.com/BurntSushi/xgbutil` | latest | Linux X11工具 | 必需（Linux） |
| `golang.org/x/sys` | v0.15.0 | 系统调用（官方） | 必需（Windows） |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML配置解析 | 必需 |

### 标准库使用
- `encoding/json` - JSON数据处理
- `log/slog` - 结构化日志
- `flag` - 命令行参数
- `time` - 定时器
- `os/signal`, `syscall` - 信号处理
- `sync`, `context` - 并发控制
- `os`, `io` - 文件操作

## 活跃状态检测逻辑

### 核心策略
采用**查询空闲时间**方式，而非监听所有输入事件，以最小化开销。

### 检测流程
```
每1秒执行一次检测:
1. 查询当前前台窗口
2. 查询系统空闲时间（距离上次输入的秒数）
3. 如果空闲时间 < 5分钟:
   - 当前窗口活跃时间 +1秒
4. 如果空闲时间 >= 5分钟:
   - 所有窗口暂停计时
5. 每60秒批量写入数据库一次
```

### 平台实现
- **Linux**: 使用 XScreenSaverInfo.idle 获取空闲时间
- **Windows**: 使用 GetLastInputInfo() 获取空闲时间

### 特殊情况处理
1. **窗口切换**: 按窗口切换分段记录，切换时切换活跃目标
2. **系统休眠**: 休眠期间不计入任何应用的使用时间
3. **屏幕锁定**: 锁定后停止所有计时
4. **空闲超时**: 5分钟无输入后停止计时

### 记录方式
- 检测粒度: 秒级
- 记录粒度: 分钟级（每分钟持久化一次）
- 记录单位: 累计时长（秒）

### 性能目标
- 内存占用: < 20MB
- CPU占用: < 1%（空闲时）
- 启动时间: < 1秒

## 项目结构

```
actime/
├── cmd/
│   ├── actime/
│   │   └── main.go           # CLI工具入口
│   └── actimed/
│       └── main.go           # 守护进程入口
├── internal/
│   ├── core/
│   │   ├── tracker.go        # 核心跟踪逻辑
│   │   ├── timer.go          # 计时逻辑
│   │   └── types.go          # 核心数据类型
│   ├── platform/
│   │   ├── interface.go      # 平台接口定义
│   │   ├── linux_x11.go      # Linux实现（build tag）
│   │   └── windows.go        # Windows实现（build tag）
│   ├── storage/
│   │   ├── db.go             # 数据库操作
│   │   └── models.go         # 数据模型
│   ├── config/
│   │   └── config.go         # 配置管理
│   └── service/
│       └── service.go        # 服务管理
├── pkg/
│   └── logger/
│       └── logger.go         # 日志封装
├── configs/
│   └── config.yaml           # 配置文件模板
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 配置文件

### 格式
YAML

### 配置项
```yaml
database:
  path: ~/.actime/actime.db

monitor:
  check_interval: 1s          # 检查间隔
  activity_window: 5m         # 活动窗口（空闲超时）
  idle_timeout: 10m           # 空闲超时（可选）

logging:
  level: info                 # 日志级别
  file: ~/.actime/actime.log  # 日志文件路径
  max_size_mb: 100            # 最大文件大小
  max_age_days: 30            # 保留天数

export:
  output_dir: ~/.actime/exports  # 导出目录
```

## 命令行接口

```bash
# 服务管理
actimed start      # 启动守护进程
actimed stop       # 停止守护进程
actimed restart    # 重启守护进程
actimed status     # 查看服务状态

# 数据查询
actime stats       # 查看统计信息
actime export      # 导出数据
  --format csv/json
  --output file
  --date range

# 配置管理
actime config      # 查看配置
actime logs        # 查看日志
```

## 数据库设计

### 表结构

**sessions 表** - 使用会话记录
```sql
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app_name TEXT NOT NULL,           -- 应用名称
    window_title TEXT,                -- 窗口标题
    start_time DATETIME NOT NULL,     -- 开始时间
    end_time DATETIME,                -- 结束时间
    duration_seconds INTEGER NOT NULL, -- 持续时长（秒）
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_app_name ON sessions(app_name);
CREATE INDEX idx_sessions_start_time ON sessions(start_time);
```

**daily_stats 表** - 每日统计
```sql
CREATE TABLE daily_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app_name TEXT NOT NULL,
    date DATE NOT NULL,
    total_seconds INTEGER NOT NULL,
    UNIQUE(app_name, date)
);

CREATE INDEX idx_daily_stats_date ON daily_stats(date);
```

## 开发计划

### 阶段1: 基础框架
- [ ] 初始化Go项目
- [ ] 创建目录结构
- [ ] 实现配置管理模块
- [ ] 实现日志模块
- [ ] 实现核心数据类型

### 阶段2: 平台检测
- [ ] 定义平台接口
- [ ] 实现Linux窗口检测（X11）
- [ ] 实现Linux空闲时间检测
- [ ] 实现Windows窗口检测
- [ ] 实现Windows空闲时间检测

### 阶段3: 核心跟踪逻辑
- [ ] 实现前台窗口检测
- [ ] 实现空闲时间检测
- [ ] 实现5分钟活动窗口逻辑
- [ ] 实现会话管理
- [ ] 处理窗口切换

### 阶段4: 数据存储
- [ ] 设计数据库Schema
- [ ] 实现数据库初始化
- [ ] 实现会话数据CRUD
- [ ] 实现批量写入优化
- [ ] 实现数据库迁移

### 阶段5: 服务管理
- [x] 实现跨平台服务封装
- [x] 实现启动/停止逻辑
- [x] 实现状态监控
- [x] 实现优雅关闭
- [x] 实现PID文件管理
- [x] 实现守护进程模式
- [x] 添加日志查看功能

### 阶段6: CLI工具
- [x] 实现start/stop/status命令
- [x] 实现restart命令
- [x] 实现log命令
- [x] 实现stats查询
- [x] 实现export功能
- [x] 完善错误处理和返回码

### 阶段7: 测试和优化
- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能优化
- [ ] Bug修复

### 阶段8: 打包和部署
- [ ] 跨平台构建脚本
- [ ] 安装脚本
- [ ] 文档完善

## 性能优化策略

### 内存优化
- 使用对象池 `sync.Pool` 复用对象
- 避免频繁的内存分配
- 使用值类型而非指针（小对象）

### CPU优化
- 批量写入减少数据库IO
- 合理设置检查间隔（1秒）
- 使用 `time.Sleep` 而非忙等待

### IO优化
- 使用缓冲写入
- 定期批量提交数据（每分钟）
- 异步日志写入

### 数据库优化
- 使用事务批量插入
- 合理设计索引
- 定期VACUUM清理

## 注意事项

### Linux平台
- 需要X11环境
- Wayland支持需要额外处理
- 需要DISPLAY环境变量

### Windows平台
- 需要管理员权限（某些功能）
- 支持Windows 7+

### 跨平台编译
- 使用build tag区分平台
- CGO依赖（sqlite使用纯Go版本避免CGO）
- 交叉编译测试

## 服务管理实现

### PID文件管理
- **位置**: `/tmp/actime.pid`
- **作用**: 防止重复启动，跟踪服务进程
- **实现**:
  - 启动时检查PID文件是否存在
  - 如果存在，检查进程是否仍在运行
  - 如果进程已停止，清理过期PID文件
  - 如果进程仍在运行，拒绝启动并返回错误

### 守护进程模式
- **实现方式**: 使用 `exec.Command` + `syscall.SysProcAttr{Setsid: true}`
- **特点**:
  - 从终端分离，独立运行
  - 不受父进程退出影响
  - 真正的后台服务

### 信号处理
- **SIGTERM**: 优雅关闭
  - 停止tracker
  - 刷新所有会话数据
  - 关闭数据库连接
  - 删除PID文件
- **SIGINT**: 同SIGTERM

### 日志管理
- **日志文件**: `~/.actime/actime.log`
- **轮转策略**:
  - 最大文件大小: 100MB
  - 最大保留天数: 30天
  - 最大备份数: 10个
- **查看方式**: 使用 `actimed log` 命令查看最近50条日志

### 错误处理
- **返回码**:
  - 成功: 0
  - 失败: 1
- **常见错误**:
  - "service is already running" - 服务已在运行
  - "service is not running" - 服务未运行
  - "failed to read PID file" - PID文件读取失败
  - "failed to send stop signal" - 停止信号发送失败