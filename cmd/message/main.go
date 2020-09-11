package main

import (
	"github.com/urfave/cli"
	"log"
	"os"
)


const usage  = "长连接socket服务，实现接收logic推送来的消息，合并发送到socket客户端"

func main()  {
	app := cli.NewApp()
	app.Name = "message-server"
	app.Usage = usage
	
	app.Commands = []cli.Command {
		runCommand,
	}
	app.Before = func(context *cli.Context) error {
		log.SetOutput(os.Stdout)
		return nil
	}
	
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
