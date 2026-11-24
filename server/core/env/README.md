# env 模块

配置模块：初始化 `.env` 配置文件并补全缺失的配置项（不覆盖已有值），基于 `github.com/1szt/go-pkg/env`。

# 使用方法

env.Start() 初始化配置（须在 motd 之后、其他模块之前调用）

# 配置文件

默认路径 `data/.env`，可用环境变量覆盖任意项。`env.Add` 不存在则创建、存在则保留原值。

# 配置项清单

## 服务

| 键 | 默认值 | 说明 |
| --- | --- | --- |
| LFDD_PORT | 9081 | 端口 |
| LFDD_SERVER_IP | "" | 服务器 IP |
| LFDD_DEBUG | false | 调试模式（true 时 gin 输出路由注册等调试日志） |
| LFDD_SERVER_INSTALL | false | 是否已完成初始化安装 |

## HTTPS

| 键 | 默认值 | 说明 |
| --- | --- | --- |
| LFDD_SERVER_HTTPS | true | 是否启用 HTTPS |
| LFDD_SERVER_HTTPS_AUTO | true | 是否自签名证书 |
| LFDD_SERVER_CERT | "" | HTTPS 证书路径 |
| LFDD_SERVER_KEY | "" | HTTPS 私钥路径 |

## 日志

| 键 | 默认值 | 说明 |
| --- | --- | --- |
| LFDD_LOG_PATH | data/log/lfdd.log | 日志文件路径（空则仅输出控制台） |

## 数据库

| 键 | 默认值 | 说明 |
| --- | --- | --- |
| LFDD_DB_TYPE | "" | 数据库类型（sqlite/pgsql/mysql） |
| LFDD_DB_PATH | "" | SQLite 数据库文件路径 |
| LFDD_DB_HOST | "" | 数据库主机地址 |
| LFDD_DB_PORT | "" | 数据库端口 |
| LFDD_DB_NAME | "" | 数据库名 |
| LFDD_DB_USER | "" | 数据库用户名 |
| LFDD_DB_PASSWORD | "" | 数据库密码 |

# 读取方式

其他模块通过 `github.com/1szt/go-pkg/env` 的 `env.Get(key)` 读取、`env.Set(key, value)` 写入（apiinstall 初始化时使用）。注意 `logx.Start()` 在本模块之前执行，直接从 `data/.env` 读 `LFDD_LOG_PATH`，不依赖 `env.Add` 补全。
