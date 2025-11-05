package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	WASM     WASMConfig     `mapstructure:"wasm"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
}

type WASMConfig struct {
	ModulesPath string `mapstructure:"modules_path"`
	MaxMemory   int64  `mapstructure:"max_memory"`
}

func Load() (*Config, error) {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("wasm.modules_path", "./modules")
	viper.SetDefault("wasm.max_memory", 1073741824) // 1GB

	viper.AutomaticEnv()
	
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	
	return &cfg, nil
}