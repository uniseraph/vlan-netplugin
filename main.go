package main

import (
	"github.com/docker/libkv/store/zookeeper"
	"github.com/codegangsta/cli"
	"os"
	"github.com/Sirupsen/logrus"
)


var Version string

func init() {


	zookeeper.Register()

}


func main()  {

	app := cli.NewApp()
	app.Usage = "Network driver for Docker"
	app.Version = Version

	app.Author = "zhengtao.wuzt"
	app.Email = "zhengtao.wuzt@gmail.com"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "log-level, l",
			Value:  "info",
			EnvVar: "LOG_LEVEL",
			Usage:  "Log level (options: debug, info, warn, error, fatal, panic)",
		},
	}

	app.Before = func(c *cli.Context) error {
		logrus.SetOutput(os.Stderr)
		level, err := logrus.ParseLevel(c.String("log-level"))
		if err != nil {
			logrus.Fatalf(err.Error())
		}
		logrus.SetLevel(level)
		return nil
	}


	app.Action = func(c *cli.Context) error {

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}

}