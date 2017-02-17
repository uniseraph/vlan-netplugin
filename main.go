package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/zookeeper"
	"github.com/omega/vlan-netplugin/driver"
	"github.com/opencontainers/runc/libcontainer/user"
	"net/url"
	"os"
	"strings"
)

var Version string

func init() {

	zookeeper.Register()

}

func main() {

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

	app.Commands = []cli.Command{
		{
			Name:  "start",
			Usage: "start a vlan netplugin ",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "cluster-store",
					EnvVar: "NP_CLUSTER_STORE",
					Value:  "zk://localhost:2181",
					Usage:  "Set the cluster store",
				},
				cli.StringFlag{
					Name:   "parent-eth",
					EnvVar: "NP_ETH",
					Value:  "eth0",
					Usage:  "Set the parent eth for vlan device",
				},
			},
			Action: func(c *cli.Context) error {

				clusterStore := c.String("cluster-store")
				url, err := url.Parse(clusterStore)
				if err != nil {
					return err
				}

				s, err := libkv.NewStore(store.Backend(url.Scheme), strings.Split(url.Host, ","), nil)
				if err != nil {
					return err
				}

				d, err := driver.New(driver.DriverOption{Store: s, Prefix: url.Path, ParentEth: c.String("parent-eth")})
				if err != nil {
					return err
				}

				group, err := user.CurrentGroup()
				if err != nil {
					return nil
				}

				if err := network.NewHandler(d).ServeUnix("vlan", group.Gid); err != nil {
					return err
				}

				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}

}
