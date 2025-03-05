package config

import (
	"main/utility"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DataBase string `mapstructure:"dbname"`
		SSLMode  string `mapstructure:"sslmode"`
	} `mapstructure:"connect_db_params"`
	Server struct {
		Port int    `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"server_address"`
	JWT struct {
		Key string `mapstructure:"key"`
	} `mapstructure:"jwt"`
	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"connect_redis_params"`
}

var AppConfig Config

func LoadConfig() {
	// Load logger
	logger := utility.AppLogger.Logger
	// Load configuration
	// viper.AddConfigPath("./config")
	viper.AddConfigPath("/app/config")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	err := viper.ReadInConfig()
	if err != nil {
		logger.Error("Fatal error config file", zap.Error(err))
	}

	logger.Info("Config loaded successfully")

	err = viper.Unmarshal(&AppConfig)
	if err != nil {
		logger.Error("Unable to decode into struct", zap.Error(err))
	}
	logger.Info("Config loaded successfully", zap.Any("Config", AppConfig))
}
