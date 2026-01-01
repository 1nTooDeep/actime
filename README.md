# Actime

Actime 是一个跨平台（Windows + Linux）的后台进程，用于准确统计用户在前台软件上的真实活跃时间。

## 特性

- 🎯 **准确统计**: 基于前台窗口和用户输入活动，准确记录应用使用时长
- 🚀 **高性能**: 低内存占用（< 20MB），低CPU占用（< 1%）
- 🔒 **隐私保护**: 仅记录使用时长，不记录输入内容，数据仅本地存储
- 📊 **数据导出**: 支持CSV和JSON格式导出
- 📈 **数据可视化**: 支持生成HTML可视化报告，包含5种图表类型
- 🔄 **后台运行**: 支持作为系统服务运行，支持自启动
- 💻 **跨平台**: 支持Linux（X11）和Windows 7+

## 项目结构

```
actime/
├── cmd/                    # 命令行工具
│   ├── actime/            # CLI工具入口
│   │   └── main.go
│   └── actimed/           # 守护进程入口
│       └── main.go
├── internal/              # 内部包
│   ├── core/              # 核心逻辑
│   │   ├── tracker.go     # 核心跟踪逻辑
│   │   ├── timer.go       # 计时逻辑
│   │   └── types.go       # 核心数据类型
│   ├── platform/          # 平台相关
│   │   ├── interface.go   # 平台接口定义
│   │   ├── linux_x11.go   # Linux实现（build tag）
│   │   └── windows.go     # Windows实现（build tag）
│   ├── storage/           # 数据存储
│   │   ├── db.go          # 数据库操作
│   │   └── models.go      # 数据模型
│   ├── config/            # 配置管理
│   │   └── config.go
│   └── service/           # 服务管理
│       └── service.go
├── pkg/                   # 公共包
│   └── logger/            # 日志
│       └── logger.go
├── configs/               # 配置文件
│   └── config.yaml        # 配置模板
├── scripts/               # 脚本
│   ├── install.sh         # Linux安装脚本
│   └── install.ps1        # Windows安装脚本
├── docs/                  # 文档
│   ├── technical-decisions.md       # 技术决策
│   ├── requirements-decisions.md    # 需求决策
│   ├── progress.md                 # 项目进度
│   ├── limitations.md              # 已知限制
│   └── development-guidelines.md   # 开发规范
├── go.mod                 # Go模块定义
├── go.sum                 # Go依赖锁定
├── Makefile               # 构建脚本
└── README.md              # 项目说明
```

## 快速开始

### 环境要求

- Go 1.21+
- Linux: X11环境
- Windows: Windows 7+

### 安装

#### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/weii/actime.git
cd actime

# 编译
make build

# 安装
make install
```

#### 使用预编译二进制文件

下载对应平台的二进制文件，解压后直接运行。

### 配置

首次运行会自动创建配置文件 `~/.actime/config.yaml`：

```yaml
database:
  path: ~/.actime/actime.db

monitor:
  check_interval: 1s
  activity_window: 5m
  idle_timeout: 10m

logging:
  level: info
  file: ~/.actime/actime.log
  max_size_mb: 100
  max_age_days: 30

export:
  output_dir: ~/.actime/exports
  default_format: csv

app_mapping:
  process_names:
    chrome: "Google Chrome"
    firefox: "Mozilla Firefox"
    code: "VS Code"
  browsers:
    - chrome
    - firefox
```

**配置说明**:
- `database.path`: 数据库文件路径（支持 `~` 符号自动展开）
- `monitor.check_interval`: 检测间隔（默认1秒）
- `monitor.activity_window`: 活跃时间窗口（默认5分钟）
- `monitor.idle_timeout`: 空闲超时（默认10分钟）
- `logging.level`: 日志级别（debug/info/warn/error）
- `logging.file`: 日志文件路径（支持 `~` 符号自动展开）
- `export.output_dir`: 导出目录（支持 `~` 符号自动展开）
- `export.default_format`: 默认导出格式（csv/json）
- `app_mapping.process_names`: 进程名称映射（不区分大小写）
- `app_mapping.browsers`: 浏览器进程名称列表

### 使用

#### 启动服务

```bash
# 启动守护进程
actimed start

# 查看状态
actimed status

# 查看日志
actimed log

# 停止服务
actimed stop

# 重启服务
actimed restart
```

**服务管理特性**:
- ✅ 防止重复启动（如果服务已在运行，会返回错误）
- ✅ 守护进程模式（服务在后台持续运行）
- ✅ 优雅关闭（停止时会保存所有数据）
- ✅ 状态监控（实时显示服务运行状态和PID）
- ✅ 日志查看（显示最近50条日志记录）
- ✅ 正确的返回码（成功=0，失败=1）

#### 查看统计

```bash
# 查看今日统计
actime stats

# 查看最近7天统计
actime stats --days 7
```

#### 导出数据

```bash
# 导出为CSV
actime export --format csv --output report.csv

# 导出为JSON
actime export --format json --output report.json

# 按日期范围导出
actime export --format csv --start 2026-01-01 --end 2026-01-31
```

#### 数据可视化

```bash
# 生成可视化报告（默认最近7天）
actime visualize

# 指定日期范围
actime visualize --start 2026-01-01 --end 2026-01-31

# 自定义输出文件
actime visualize --output my-report.html

# 生成后自动在浏览器中打开
actime visualize --open

# 查看最近30天的数据
actime visualize --days 30
```

**可视化报告包含**:
- 📊 总览：应用使用排行（柱状图 + 饼图）
- 📈 趋势：每日使用变化（折线图）
- ⏰ 节律：时间段分布（热力图）
- 🧩 构成：应用时间结构（树状图）

## 工作原理

Actime 通过以下方式统计应用使用时长：

1. **前台窗口检测**: 每秒检测当前活动的应用程序窗口
2. **空闲时间检测**: 查询系统空闲时间（距离上次输入的时间）
3. **活跃判断**: 如果空闲时间 < 5分钟，则认为用户活跃
4. **时间记录**: 记录每个应用的累计活跃时长
5. **数据持久化**: 每分钟批量写入数据库
6. **数据可视化**: 使用 ECharts 生成交互式 HTML 报告，支持多种图表类型

### 平台实现

- **Linux**: 使用X11协议获取窗口信息和空闲时间
- **Windows**: 使用Win32 API获取窗口信息和空闲时间

详细技术说明请参考 [技术决策文档](docs/technical-decisions.md)。

## 性能指标

- 内存占用: < 20MB
- CPU占用: < 1%（空闲时）
- 启动时间: < 1秒
- 检测间隔: 1秒
- 数据持久化间隔: 60秒
- 日志轮转: 自动轮转，最大100MB，保留30天

## 已知限制

- Linux仅支持X11环境，不支持Wayland
- Windows某些功能可能需要管理员权限
- 不记录输入内容（隐私保护）
- 不支持多用户同时监控

详细限制说明请参考 [已知限制文档](docs/limitations.md)。

## 开发

### 环境设置

```bash
# 安装依赖
go mod download

# 运行测试
make test

# 代码检查
make lint

# 格式化代码
make fmt
```

### 开发规范

请参考 [开发规范文档](docs/development-guidelines.md)。

### 项目进度

查看 [项目进度文档](docs/progress.md) 了解当前开发状态。

## 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feat/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feat/amazing-feature`)
5. 创建 Pull Request

## 许可证

[MIT License](LICENSE)

## 联系方式

- 项目主页: https://github.com/weii/actime
- 问题反馈: https://github.com/weii/actime/issues

## 致谢

感谢所有贡献者的支持！

## 常见问题

### Q: Actime 会记录我的输入内容吗？
A: 不会。Actime 仅记录应用使用时长，不记录任何输入内容。

### Q: 数据会上传到云端吗？
A: 不会。所有数据仅存储在本地，不上传到任何服务器。

### Q: 如何在Wayland环境下使用？
A: 当前版本不支持Wayland。可以使用XWayland兼容层，或等待未来版本支持。

### Q: 数据库文件在哪里？
A: 默认在 `~/.actime/actime.db`，可以在配置文件中修改。

### Q: 如何卸载？
A: 停止服务后，删除配置文件和数据目录即可。

### Q: 服务启动失败怎么办？
A: 检查日志文件 `~/.actime/actime.log` 查看详细错误信息。可以使用 `actimed log` 命令快速查看。

### Q: 如何查看服务是否在运行？
A: 使用 `actimed status` 命令，会显示服务状态和进程ID。

### Q: 可以启动多个服务实例吗？
A: 不可以。服务使用PID文件管理，会自动检测并防止重复启动。

### Q: 可视化报告支持哪些图表类型？
A: 支持柱状图、饼图、折线图、热力图和树状图，共5种图表类型。

### Q: 如何自定义应用名称的显示？
A: 在配置文件的 `app_mapping.process_names` 中添加进程名称映射，例如：
```yaml
app_mapping:
  process_names:
    chrome.exe: "Google Chrome"
    code.exe: "VS Code"
```

### Q: 可视化报告需要联网吗？
A: 报告使用 CDN 加载 ECharts 库，需要联网才能正常显示图表。如需离线使用，可下载 ECharts 库到本地并修改 HTML 文件。

更多问题请查看 [Issues](https://github.com/weii/actime/issues)。