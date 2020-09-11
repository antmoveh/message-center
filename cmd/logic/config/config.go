package config

import (
	"github.com/spf13/viper"
	"log"
	"os"
)

type MessageServerConfig struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
}

// 程序配置
type Config struct {
	ServicePort                      int                   `json:"servicePort"`
	ServiceReadTimeout               int                   `json:"serviceReadTimeout"`
	ServiceWriteTimeout              int                   `json:"serviceWriteTimeout"`
	MessageServerList                []MessageServerConfig `json:"messageServerList"`
	MessageServerMaxConnection       int                   `json:"messageServerMaxConnection"`
	MessageServerTimeout             int                   `json:"messageServerTimeout"`
	MessageServerIdleTimeout         int                   `json:"messageServerIdleTimeout"`
	MessageServerDispatchWorkerCount int                   `json:"messageServerDispatchWorkerCount"`
	MessageServerDispatchChannelSize int                   `json:"messageServerDispatchChannelSize"`
	MessageServerMaxPendingCount     int                   `json:"messageServerMaxPendingCount"`
	MessageServerPushRetry           int                   `json:"messageServerPushRetry"`
}

var GlobalLogicConfig *Config

func LoadConfig() error {
	configPath := os.Getenv("CONFIG")
	if configPath == "" {
		msc := []MessageServerConfig{{
			Hostname: "localhost",
			Port:     7788,
		}}
		c := Config{
			ServicePort:                      7799,
			ServiceReadTimeout:               2000,
			ServiceWriteTimeout:              2000,
			MessageServerList:                msc,
			MessageServerMaxConnection:       4,
			MessageServerTimeout:             300,
			MessageServerIdleTimeout:         60,
			MessageServerDispatchWorkerCount: 8,
			MessageServerDispatchChannelSize: 1000,
			MessageServerMaxPendingCount:     20,
			MessageServerPushRetry:           3,
		}
		GlobalLogicConfig = &c
		return nil
	}
	config := viper.New()
	config.AddConfigPath(configPath)
	config.SetConfigName("config")
	config.SetConfigType("json")
	if err := config.ReadInConfig(); err != nil {
		log.Fatal(err)
	}
	var c Config
	if err := config.Unmarshal(&c); err != nil {
		log.Fatal(err)
	}
	GlobalLogicConfig = &c
	return nil
}
