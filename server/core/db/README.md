# DB 模块

DB 模块负责数据库连接管理，支持 SQLite、MySQL、PostgreSQL（ORM 底层为 gorm）。

SQLite 连接前会自动创建 SBCC_DB_PATH 所在目录（父目录不存在时报 "unable to open database file"）。

# 使用方法

db.DB 全局数据库实例

db.Start() 启动时检查初始化状态，未初始化则跳过连接
db.Stop() 关闭数据库连接
db.Restart() 强制重连（关闭旧连接 → 重新读取配置 → 连接）
db.Reload() 智能重连（配置无变化且已连接则跳过，否则重连）
db.Status() 获取数据库状态

# Status 返回结构体

Type - 数据库类型（sqlite/mysql/postgresql）
Host - 主机地址
Port - 端口
Name - 数据库名
Status - 连接状态（connected/disconnected）
