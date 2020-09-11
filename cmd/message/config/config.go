package config

import (
	"github.com/spf13/viper"
	"log"
	"os"
)

// socket服务启动配置
type Config struct {
	WsPort               int    `json:"wsPort"`
	WsReadTimeout        int    `json:"wsReadTimeout"`
	WsWriteTimeout       int    `json:"wsWriteTimeout"`
	WsInChannelSize      int    `json:"wsInChannelSize"`
	WsOutChannelSize     int    `json:"wsOutChannelSize"`
	WsHeartbeatInterval  int    `json:"wsHeartbeatInterval"`
	MaxMergerDelay       int    `json:"maxMergerDelay"`
	MaxMergerBatchSize   int    `json:"maxMergerBatchSize"`
	MergerWorkerCount    int    `json:"mergerWorkerCount"`
	MergerChannelSize    int    `json:"mergerChannelSize"`
	ServicePort          int    `json:"servicePort"`
	ServiceReadTimeout   int    `json:"serviceReadTimeout"`
	ServiceWriteTimeout  int    `json:"serviceWriteTimeout"`
	ServerPem            string `json:"serverPem"`
	ServerKey            string `json:"serverKey"`
	BucketCount          int    `json:"bucketCount"`
	MaxJoinRoom          int    `json:"maxJoinRoom"`
	DispatchChannelSize  int    `json:"dispatchChannelSize"`
	DispatchWorkerCount  int    `json:"dispatchWorkerCount"`
	BucketJobChannelSize int    `json:"bucketJobChannelSize"`
	BucketJobWorkerCount int    `json:"bucketJobWorkerCount"`
}

var GlobalServerConfig *Config

func LoadConfig() error {
	configPath := os.Getenv("CONFIG")
	if configPath == "" {
		c := Config{
			WsPort:               7777,
			WsReadTimeout:        2000,
			WsWriteTimeout:       2000,
			WsInChannelSize:      1000,
			WsOutChannelSize:     1000,
			WsHeartbeatInterval:  60,
			MaxMergerDelay:       300,
			MaxMergerBatchSize:   100,
			MergerWorkerCount:    8,
			MergerChannelSize:    1000,
			ServicePort:          7788,
			ServiceReadTimeout:   2000,
			ServiceWriteTimeout:  2000,
			ServerPem:            "",
			ServerKey:            "",
			BucketCount:          16,
			MaxJoinRoom:          5,
			DispatchChannelSize:  1000,
			DispatchWorkerCount:  16,
			BucketJobChannelSize: 1000,
			BucketJobWorkerCount: 2,
		}
		GlobalServerConfig = &c
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
	GlobalServerConfig = &c
	return nil
}
