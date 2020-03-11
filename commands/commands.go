package commands

import (
	"github.com/Assetsadapter/beam-adapter/beam"
	"github.com/astaxie/beego/config"
	"github.com/blocktree/openwallet/log"
	"gopkg.in/urfave/cli.v1"
)

var (
	// 通信节点命令
	Commands = []cli.Command{
		CmdVersion,
		{
			//运行钱包服务
			Name:      "walletserver",
			Usage:     "start the wallet server",
			ArgsUsage: "",
			Action:    walletserver,
			Category:  "BEAM-SERVER COMMANDS",
		},
	}
)

func getWalleManager(c *cli.Context) *beam.WalletManager {
	var (
		err error
	)

	conf := c.GlobalString("conf")
	cfg, err := config.NewConfig("ini", conf)
	if err != nil {
		return nil
	}

	wm := beam.NewWalletManager()
	wm.LoadAssetsConfig(cfg)

	return wm
}

//walletserver 钱包服务
func walletserver(c *cli.Context) error {
	var (
		endRunning = make(chan bool, 1)
	)
	if wm := getWalleManager(c); wm != nil {
		log.Error("doing something")

		//err := wm.StartSummaryWallet()
		//if err != nil {
		//	log.Error("unexpected error: ", err)
		//	return err
		//}
		//
		<-endRunning
	}

	return nil
}
