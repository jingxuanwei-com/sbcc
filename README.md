# SBCC - SB 控制中心

> **SBCC** = **SB** **C**ontrol **C**enter = **SB 控制中心**

一个基于 Go 语言构建的高性能、模块化控制中心后端框架。

---

## 📁 项目结构

```
sbcc/
├── .github/
│   └── workflows/    # GitHub Actions 工作流
│       └── docker.yml
├── .vscode/          # VS Code 配置文件
├── modbus/           # 通用模块库
│   ├── chi/          # Chi 框架 Web 引擎底座
│   ├── gin/          # Gin 框架示例模块
│   ├── gorm/         # GORM ORM 数据库层
│   ├── sql/          # 原生 SQL 数据库连接层
│   ├── sqlx/         # SQLX 增强数据库层
│   ├── env/          # 环境配置系统
│   ├── home/         # 主页模块
│   ├── podman/       # Podman 控制模块
│   ├── sub/          # 订阅 API 模块
│   ├── main/         # 程序入口
│   ├── go.mod        # Go 依赖管理
│   └── go.sum        # 依赖校验文件
├── .gitignore        # Git 忽略配置
├── LICENSE           # 许可证文件
├── README.md         # 项目说明文档
├── Dockerfile        # Docker 构建文件
├── build.bat         # Windows 构建脚本
└── go.work           # Go 工作区配置
```

---

## 📋 模块说明

### 🔧 框架层（Framework）

| 目录 | 作用 | 状态 |
|------|------|------|
| `modbus/chi/` | **Web 引擎底座**，使用 Chi 路由库，提供中间件、限流、日志等基础能力 | ✅ 核心 |
| `modbus/gin/` | **Gin 混合示例**，演示如何在 Chi 主路由上挂载 Gin 子应用 | 🧪 实验 |
| `modbus/gorm/` | **GORM ORM 层**，提供对象关系映射能力，支持事务、预加载等 | ✅ 核心 |
| `modbus/sql/` | **原生 SQL 连接层**，支持 SQLite/MySQL/PostgreSQL，带自动重连机制 | ✅ 核心 |
| `modbus/sqlx/` | **SQLX 增强层**，在原生连接基础上包装，支持命名参数等高级特性 | ✅ 扩展 |

### 🎯 业务层（Business）

| 目录 | 作用 | 状态 |
|------|------|------|
| `modbus/home/` | **主页模块**，处理根路径 `/` 的请求，演示如何挂载子路由 | ✅ 业务 |
| `modbus/podman/` | **Podman 控制模块**，提供容器管理界面 | ✅ 业务 |
| `modbus/sub/` | **订阅 API 模块**，提供 Clash 配置订阅服务，支持流量统计、到期时间等 | ✅ 业务 |

### ⚙️ 基础设施（Infrastructure）

| 目录 | 作用 | 状态 |
|------|------|------|
| `modbus/env/` | **环境配置系统**，自动创建 `.env` 文件，支持环境变量覆盖 | ✅ 核心 |
| `modbus/main/` | **程序入口**，负责启动所有模块 | ✅ 核心 |

---

## 🚀 命名规范

### 包引用规范

```go
import (
    chi "github.com/go-chi/chi/v5"  // Chi 框架包（别名）
    web "modbus/chi"                 // modbus 模块（别名）
    db "modbus/sql"                  // 数据库模块
)
```

### 变量命名规范

所有数据库模块统一使用 `DB` 变量名：

```go
sql.DB    // 原生数据库连接
sqlx.DB   // sqlx 增强连接
gorm.DB   // GORM ORM 实例
```

### 模块启动规范

所有模块遵循统一的 `.Run()` 接口：

```go
func main() {
    // Web 引擎启动
    web.Run()

    // 数据库模块启动
    db.Run()
    sqlx.Run()
    gorm.Run()

    // 功能模块启动
    home.Run()
    podman.Run()
    sub.Run()

    select {} // 阻塞主进程
}
```

---

## 🔥 核心特性

- **多数据库支持**: SQLite / MySQL / PostgreSQL 一键切换
- **三层数据库架构**: 原生 SQL → SQLX → GORM，按需选择
- **Chi + Gin 混合**: 支持在同一个项目中使用两个框架
- **自动配置**: `.env` 文件自动生成，无需手动创建
- **优雅启动**: 端口占用检测、模块状态反馈
- **高可用设计**: 数据库自动重连机制
- **模块化架构**: 新增功能只需添加 `.Run()` 方法

---

## 🚀 快速开始

### 本地运行

```bash
# 启动应用
go run modbus/main/run.go
```

启动后访问: http://localhost:9081

### Docker 运行

```bash
# 拉取镜像
docker pull ghcr.io/你的用户名/sbcc:latest

# 运行容器
docker run -d \
  --name sbcc \
  -p 9081:9081 \
  -v $(pwd)/data:/app/data \
  ghcr.io/你的用户名/sbcc:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  sbcc:
    image: ghcr.io/你的用户名/sbcc:latest
    container_name: sbcc
    ports:
      - "9081:9081"
    volumes:
      - ./data:/app/data
    restart: unless-stopped
```

---

## 🔄 CI/CD 自动构建

本项目使用 **GitHub Actions** 实现自动构建和发布。

### 构建流程

| 事件 | 触发条件 | 操作 |
|------|---------|------|
| `push` to `main` | 代码合并到主分支 | 构建并推送 `latest` 标签镜像 |
| `push tag `v*` | 发布版本标签 | 构建并推送 `v*.*.*` 和 `latest` 标签镜像 |
| `pull_request` | PR 创建/更新 | 构建测试镜像（不推送） |

### GitHub Packages 镜像

| 标签 | 说明 | 拉取命令 |
|------|------|---------|
| `latest` | 最新稳定版 | `docker pull ghcr.io/你的用户名/sbcc:latest` |
| `v1.0.0` | 指定版本 | `docker pull ghcr.io/你的用户名/sbcc:v1.0.0` |
| `sha-abc1234` | Commit ID | `docker pull ghcr.io/你的用户名/sbcc:sha-abc1234` |

---

## 📝 配置说明

启动后会自动创建 `data/.env` 文件，主要配置项：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `WEB_PORT` | 9081 | Web 服务端口 |
| `DB_TYPE` | sqlite | 数据库类型 (sqlite/mysql/pgsql) |
| `DB_NAME` | data/sbcc.db | 数据库名/文件路径 |
| `DB_HOST` | 127.0.0.1 | 数据库主机 |
| `DB_PORT` | 5432 | 数据库端口 |
| `DB_USER` | postgres | 数据库用户名 |
| `DB_PASSWORD` | your_password | 数据库密码 |

---

## 🛠️ 技术栈

- **语言**: Go 1.26+
- **Web 框架**: [Chi](https://github.com/go-chi/chi)
- **ORM**: [GORM](https://gorm.io/)
- **SQL 增强**: [SQLX](https://github.com/jmoiron/sqlx)
- **限流**: [httprate](https://github.com/go-chi/httprate)
- **容器化**: Docker

---

## 📄 许可证

Apache License 2.0
