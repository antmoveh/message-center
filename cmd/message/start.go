package main

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"log"
	"message-center/cmd/message/config"
	message_server "message-center/pkg/message-server"
	web_socket "message-center/pkg/message-server/web-socket"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var runCommand = cli.Command{
	Name:        "run",
	ShortName:   "r",
	Description: "启动socket服务",
	Usage:       "环境变量CONFIG：指定文件所在目录不要指定具体文件config.json",
	Flags:       []cli.Flag{},

	Before: func(context *cli.Context) error {
		// global.ServerConfig = config.LoadConfig()
		// runtime.GOMAXPROCS(runtime.NumCPU())
		runtime.GOMAXPROCS(4)
		return nil
	},
	Action: func(context *cli.Context) error {
		start()
		return nil
	},
}

func start() {
	logrus.Info("加载配置")
	_ = config.LoadConfig()

	logrus.Info("启动连接管理器，管理socket连接")
	err := web_socket.InitConnectManager()
	if err != nil {
		log.Fatal("初始化socket连接管理器失败：" + err.Error())
	}

	logrus.Info("启动websocket connect endpoint: 0.0.0.0:7777")
	err = web_socket.InitSocketEndpoint()
	if err != nil {
		log.Fatal("启动websocket connect endpoint失败:" + err.Error())
	}

	logrus.Info("初始化消息合并")
	err = web_socket.InitMessageMerger()
	if err != nil {
		log.Fatal("初始化消息合并服务失败：" + err.Error())
	}

	logrus.Info("启动HTTP服务：127.0.0.1:7788")
	err = message_server.InitHttpService()
	if err != nil {
		log.Fatal("启动HTTP服务失败：" + err.Error())
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		select {
		case <-sigCh:
			logrus.Info("logic系统关闭")
			message_server.HttpServerClose()
			web_socket.SocketConnectClose()
			web_socket.GlobalMessageMergeServer.MergeClose()
			web_socket.GlobalSocketConnectionManager.ConnectManagerClose()
			return
		}
	}
}
