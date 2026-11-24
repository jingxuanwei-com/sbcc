// Package config 使用 Viper 加载 data/config.yaml（YAML 格式），并反序列化为 Config。
package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 是 sbcc 的项目配置，对应 data/config.yaml。
type Config struct {
	Server      Server `mapstructure:"server"`
	DB          DB     `mapstructure:"db"`
	Debug       bool   `mapstructure:"debug"`
	Install     bool   `mapstructure:"install"`
	LogPath     string `mapstructure:"log_path"`
	CORSOrigins string `mapstructure:"cors_origins"`
	UploadPath  string `mapstructure:"upload_path"`
}

// Server 服务监听配置
type Server struct {
	IP       string `mapstructure:"ip"`
	Port     int    `mapstructure:"port"`
	HTTPS    bool   `mapstructure:"https"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DB 数据库配置
type DB struct {
	Type     string `mapstructure:"type"`
	Path     string `mapstructure:"path"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

// Cfg 全局配置，Start 成功后可用。
var Cfg *Config

// Start 读取 data/config.yaml；文件不存在时按默认值生成，失败直接退出进程。
// 环境变量以 SBCC_ 为前缀覆盖配置，如 SBCC_SERVER_PORT=9000。
func Start() {
	if err := load(); err != nil {
		log.Fatal(err)
	}
}

func load() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("data")
	viper.SetEnvPrefix("SBCC")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("config: 读取配置失败: %w", err)
		}
		if err := os.MkdirAll("data", 0755); err != nil {
			return fmt.Errorf("config: 创建 data 目录失败: %w", err)
		}
		if err := viper.SafeWriteConfig(); err != nil {
			return fmt.Errorf("config: 生成 data/config.yaml 失败: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("config: 解析配置失败: %w", err)
	}
	Cfg = &cfg
	return nil
}

// setDefaults 登记默认值，同时保证键名存在（Unmarshal 与环境变量覆盖都依赖它）。
func setDefaults() {
	viper.SetDefault("server.ip", "")
	viper.SetDefault("server.port", 9081)
	viper.SetDefault("server.https", false)
	viper.SetDefault("server.cert_file", "")
	viper.SetDefault("server.key_file", "")

	viper.SetDefault("db.type", "sqlite")
	viper.SetDefault("db.path", "data/db/sbcc.db")
	viper.SetDefault("db.host", "")
	viper.SetDefault("db.port", "")
	viper.SetDefault("db.name", "")
	viper.SetDefault("db.user", "")
	viper.SetDefault("db.password", "")

	viper.SetDefault("debug", false)
	viper.SetDefault("install", true)
	viper.SetDefault("log_path", "data/log/sbcc.log")
	viper.SetDefault("cors_origins", "*")
	viper.SetDefault("upload_path", "data/upload")
}
