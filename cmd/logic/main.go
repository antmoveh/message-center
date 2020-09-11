package main

import (
	"github.com/urfave/cli"
	"log"
	"os"
)

const usage = "实现推送消息业务逻辑层，定时更新消息，推送给消息服务"

func main() {
	app := cli.NewApp()
	app.Name = "logic-server"
	app.Usage = usage

	app.Commands = []cli.Command{
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
