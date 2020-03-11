package beam

import (
	"encoding/json"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/blocktree/openwallet/owtp"
	"strconv"
)

type Server struct {
	wm                *WalletManager
	node              *owtp.OWTPNode
	config            *WalletConfig
	disconnectHandler func(node *Server, nodeID string)           //托管节点断开连接后的通知
	connectHandler    func(node *Server, nodeInfo *TrustNodeInfo) //托管节点连接成功的通知
}

func NewServer(wm *WalletManager) (*Server, error) {

	config := wm.Config

	cert := owtp.NewRandomCertificate()

	connectCfg := owtp.ConnectConfig{}
	connectCfg.Address = config.remoteserver
	connectCfg.EnableSSL = config.enablessl
	connectCfg.ConnectType = config.connecttype
	node := owtp.NewNode(owtp.NodeConfig{
		Cert:       cert,
		TimeoutSEC: config.requesttimeout,
	})

	t := &Server{
		node:   node,
		config: config,
		wm:     wm,
	}

	node.HandleFunc("newNodeJoin", t.newNodeJoin)
	node.HandleFunc("getTransactionsByHeight", t.getTransactionsByHeight)
	node.HandleFunc("getTransaction", t.getTransaction)
	node.HandleFunc("createBatchAddress", t.createBatchAddress)
	node.HandleFunc("getWalletBalance", t.getWalletBalance)
	node.HandleFunc("getWalletAddress", t.getWalletAddress)
	node.HandleFunc("getBlockByHeight", t.getBlockByHeight)
	node.HandleFunc("getWalletStatus", t.GetWalletStatus)
	node.HandleFunc("transFCoin", t.transFCoin)
	node.HandleFunc("summaryToAddress", t.summaryToAddress)
	node.HandleFunc("validateAddress", t.validateAddress)

	node.SetCloseHandler(func(n *owtp.OWTPNode, peer owtp.PeerInfo) {
		if t.disconnectHandler != nil {
			t.disconnectHandler(t, peer.ID)
		}
	})

	return t, nil
}

//Listen 启动监听
func (server *Server) Listen() {

	//开启监听
	server.wm.Log.Infof("Transmit node IP %s start to listen [%s] connection...", server.config.remoteserver, server.config.connecttype)

	server.node.Listen(owtp.ConnectConfig{
		Address:     server.config.remoteserver,
		ConnectType: server.config.connecttype,
	})
}

//Close 关闭监听
func (server *Server) Close() {
	server.node.Close()
}

//SetConnectHandler 设置托管节点断开连接后的通知
func (server *Server) SetConnectHandler(h func(node *Server, nodeInfo *TrustNodeInfo)) {
	server.connectHandler = h
}

//SetDisconnectHandler 设置托管节点连接成功的通知
func (server *Server) SetDisconnectHandler(h func(node *Server, nodeID string)) {
	server.disconnectHandler = h
}

//checkTrustNode 检查是否授信节点
func (server *Server) checkTrustNode(nodeID string) bool {
	//判断连接的客户端NodeID是否授信
	trustNodeID := server.wm.Config.trustnodeid
	if len(trustNodeID) > 0 {
		if trustNodeID != nodeID {
			log.Warningf("The Joining Node: %s is not trusted", nodeID)
			server.node.ClosePeer(nodeID)
			return false
		}
	}

	return true
}

//检查ip配置
func (server *Server) checkIP(IP string) bool {
	//server.config.DefaultConfig
	return true
}

/*********** 本地路由方法实现 ***********/

func (server *Server) newNodeJoin(ctx *owtp.Context) {
	server.wm.Log.Infof("node joining:%s", ctx.RemoteAddress)
	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	if server.connectHandler != nil {
		var nodeInfo TrustNodeInfo
		err := json.Unmarshal([]byte(ctx.Params().Get("nodeInfo").Raw), &nodeInfo)
		if err != nil {
			ctx.Response(nil, owtp.ErrCustomError, err.Error())
			return
		}
		server.connectHandler(server, &nodeInfo)
	}

	ctx.Response(nil, owtp.StatusSuccess, "success")
}

func (server *Server) getTransactionsByHeight(ctx *owtp.Context) {

	//server.wm.Log.Infof("Client call [getTransactionsByHeight]")

	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	height := ctx.Params().Get("height").Uint()
	txs, err := server.wm.walletClient.GetTransactionsByHeight(height)
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}

	ctx.Response(txs, owtp.StatusSuccess, "success")

	//server.wm.Log.Infof("---------------------------------------")
}

func (server *Server) getTransaction(ctx *owtp.Context) {

	server.wm.Log.Infof("Client call [getTransaction]")

	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	txid := ctx.Params().Get("txid").String()
	tx, err := server.wm.walletClient.GetTransaction(txid)
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}

	ctx.Response(tx, owtp.StatusSuccess, "success")

	server.wm.Log.Infof("---------------------------------------")
}

func (server *Server) createBatchAddress(ctx *owtp.Context) {

	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	count := ctx.Params().Get("count").Uint()
	workerSize := ctx.Params().Get("workerSize").Uint()
	server.wm.Log.Infof("Client call [createBatchAddress]")
	server.wm.Log.Infof("count: %d", count)
	server.wm.Log.Infof("workerSize: %d", workerSize)

	addrs, err := server.wm.CreateLocalWalletAddress(count, workerSize)
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}

	ctx.Response(addrs, owtp.StatusSuccess, "success")

	server.wm.Log.Infof("---------------------------------------")
}

func (server *Server) getWalletBalance(ctx *owtp.Context) {
	server.wm.Log.Infof("Client call [getWalletBalance]")

	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	balance, err := server.wm.GetLocalWalletBalance()
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}

	result := map[string]interface{}{
		"balance": balance,
	}

	server.wm.Log.Infof("balance: %+v", balance)

	ctx.Response(result, owtp.StatusSuccess, "success")

	server.wm.Log.Infof("---------------------------------------")
}

func (server *Server) getWalletAddress(ctx *owtp.Context) {
	server.wm.Log.Infof("Client call [getWalletAddress]")

	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	addrs, err := server.wm.GetLocalWalletAddress()
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}

	ctx.Response(addrs, owtp.StatusSuccess, "success")

	server.wm.Log.Infof("---------------------------------------")
}

func (server *Server) getBlockByHeight(ctx *owtp.Context) {

	//server.wm.Log.Infof("Client call [getTransactionsByHeight]")
	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	height := ctx.Params().Get("height").Uint()
	block, err := server.wm.walletClient.GetBlockByHeight(height)
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}

	ctx.Response(block, owtp.StatusSuccess, "success")

	//server.wm.Log.Infof("---------------------------------------")
}

func (server *Server) GetWalletStatus(ctx *owtp.Context) {

	//server.wm.Log.Infof("Client call [getTransactionsByHeight]")

	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	wallet, err := server.wm.walletClient.GetWalletStatus()
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}

	ctx.Response(wallet, owtp.StatusSuccess, "success")

	//server.wm.Log.Infof("---------------------------------------")
}

//转币
func (server *Server) transFCoin(ctx *owtp.Context) {

	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	toAddress := ctx.Params().Get("toAddress").String()
	toAmount := ctx.Params().Get("toAmount").String()

	rawTx := &openwallet.RawTransaction{
		To: map[string]string{
			toAddress: toAmount,
		},
		FeeRate: "",
	}
	tx, err := server.wm.TxDecoder.SubmitRawTransaction(nil, rawTx)
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}
	result := map[string]interface{}{
		"txId": tx.TxID,
	}
	ctx.Response(result, owtp.StatusSuccess, "success")
}

//汇总币到地址
func (server *Server) summaryToAddress(ctx *owtp.Context) {

	//server.wm.Log.Infof("Client call [getTransactionsByHeight]")

	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	summaryAddress := ctx.Params().Get("summaryAddress").String()
	txId, summaryAmount, feeAmount, err := server.wm.SummaryWalletProcess(summaryAddress)
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}

	ctx.Response(map[string]string{"txId": txId, "summaryAmount": summaryAmount, "feeAmount": feeAmount}, owtp.StatusSuccess, "success")

	//server.wm.Log.Infof("---------------------------------------")
}

//验证地址格式
func (server *Server) validateAddress(ctx *owtp.Context) {

	if !server.checkTrustNode(ctx.PID) {
		ctx.Response(nil, owtp.ErrDenialOfService, "the node is not trusted")
		return
	}

	address := ctx.Params().Get("address").String()
	isValid, err := server.wm.ValidateAddress(address)
	if err != nil {
		ctx.Response(nil, owtp.ErrCustomError, err.Error())
		return
	}

	ctx.Response(map[string]string{"isValid": strconv.FormatBool(isValid)}, owtp.StatusSuccess, "success")

	//server.wm.Log.Infof("---------------------------------------")
}
