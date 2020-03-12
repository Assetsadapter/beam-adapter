package commands

import (
	"fmt"
	"github.com/Assetsadapter/beam-adapter/beam"
	"github.com/astaxie/beego/config"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/owtp"
	"github.com/mr-tron/base58"
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
		{
			//随机产生一个节点数据
			Name:      "randomCert",
			Usage:     "random generate a client node cert info",
			ArgsUsage: "",
			Action:    randomGenerateClientInfo,
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

//随机生成clinet cert info
func randomGenerateClientInfo(c *cli.Context){
	cert := owtp.NewRandomCertificate()
	fmt.Printf("cert:=%s\n",base58.Encode(cert.PrivateKeyBytes()))
	nodeID := owcrypt.Hash(cert.PublicKeyBytes(), 0, owcrypt.HASH_ALG_SHA256)
	fmt.Printf("nodeId:=%s\n",base58.Encode(nodeID))
}