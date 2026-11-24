# DB 模块

数据库连接管理，支持 SQLite、MySQL、PostgreSQL，ORM 底层为 gorm。

只负责连接的建立与关闭；表结构定义与迁移在 model 模块。

## 配置

读取 `data/config.yaml` 的 `db` 段：

```yaml
db:
  type: sqlite          # sqlite / mysql / pgsql
  path: data/db/sbcc.db # SQLite 文件路径
  host: ""              # MySQL / PostgreSQL 使用
  port: ""
  name: ""
  user: ""
  password: ""
```

- SQLite：使用 `path`，连接前自动创建文件所在目录
- MySQL / PostgreSQL：使用 `host`、`port`、`name`、`user`、`password`

## 使用方法

```go
db.Start() // 按配置连接数据库；未完成初始化（install=false）时跳过
db.Stop()  // 关闭连接
```

`db.DB` 为全局 `*gorm.DB` 实例，连接成功后直接使用。

## 说明

- 连接失败只打印错误日志，不 panic
- 不提供重连 / 重载 / 状态查询；修改配置后重启进程生效
