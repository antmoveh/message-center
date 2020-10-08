package main

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"message-center/cmd/logic/config"
	"message-center/pkg/configuration"
	"message-center/pkg/email-client"
	process_message "message-center/pkg/logic-server/process-message"
	"message-center/pkg/logic-server/push"
	"message-center/pkg/mongodb"
	"message-center/pkg/mq"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var runCommand = cli.Command{
	Name:        "run",
	ShortName:   "r",
	Description: "启动接收消息服务",
	Usage:       "环境变量CONFIG：指定文件所在目录不要指定具体文件config.json",
	Flags:       []cli.Flag{},

	Before: func(context *cli.Context) error {
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

	logrus.Info("启动长连接管理服务")
	err := push.InitConnManager()
	if err != nil {
		logrus.Fatal("启动管理连接服务失败：" + err.Error())
	}

	logrus.Info("启动HTTP服务：127.0.0.1:7799")

	err = push.InitHttpService()
	if err != nil {
		logrus.Fatal("启动HTTP服务失败：" + err.Error())
	}

	logrus.Info("初始化apollo配置")
	dcl := configuration.ApolloConfigurationProvider{}
	dcl.Initialize()

	logrus.Info("初始化activemq")
	mqc := mq.MqController{}
	mqc.Initialize(&dcl)

	logrus.Info("初始化mongodb连接")
	mc := mongodb.MongoDBController{}
	mc.Initialize(&dcl)

	logrus.Info("初始化email连接")
	ec := email_client.EmailClientImpl{}
	ec.Initialize(&dcl)

	logrus.Info("初始化业务处理服务")
	pm := process_message.ProcessMessageImpl{}
	pm.Initialize(&mc, &mqc, &ec)
	pm.Run()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		select {
		case <-sigCh:
			logrus.Info("logic系统关闭")
			mc.Close()
			pm.Close()
			push.HttpServerClose()
			push.GlobalConnectManager.MessageConnectClose()
			return
		}
	}
}
