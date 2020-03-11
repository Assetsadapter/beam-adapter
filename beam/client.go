package beam

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/blocktree/openwallet/owtp"
	"time"
)

const (
	trustHostID = "openw-beam-server"
	nodeName    = "beam-client"
)

type Client struct {
	wm     *WalletManager
	node   *owtp.OWTPNode
	config *WalletConfig
}

func NewClient(wm *WalletManager) (*Client, error) {

	var (
		cert owtp.Certificate
	)

	if len(wm.Config.cert) == 0 {
		cert = owtp.NewRandomCertificate()
	} else {
		cert, _ = owtp.NewCertificate(wm.Config.cert)
	}
	node := owtp.NewNode(owtp.NodeConfig{
		Cert:       cert,
		TimeoutSEC: wm.Config.requesttimeout,
	})

	c := &Client{
		node:   node,
		config: wm.Config,
		wm:     wm,
	}

	//绑定本地路由方法
	//cli.transmitNode.HandleFunc("getTrustNodeInfo", cli.getTrustNodeInfo)

	autoReconnect := true
	//自动连接
	if autoReconnect {
		go c.autoReconnectRemoteNode()
		return c, nil
	}

	//单独连接
	err := c.connectRemoteNode()
	if err != nil {
		return nil, err
	}

	return c, nil
}

//connectTransmitNode
func (c *Client) connectRemoteNode() error {

	connectCfg := owtp.ConnectConfig{}
	connectCfg.Address = c.config.remoteserver
	connectCfg.ConnectType = c.config.connecttype
	connectCfg.EnableSSL = c.config.enablessl
	connectCfg.EnableSignature = false

	//建立连接
	_, err := c.node.Connect(trustHostID, connectCfg)
	if err != nil {
		return err
	}

	//开启协商密码
	if c.config.enablekeyagreement {
		if err = c.node.KeyAgreement(trustHostID, "aes"); err != nil {
			return err
		}
	}

	//向服务器发送连接成功
	err = c.nodeDidConnectedServer()
	if err != nil {
		return err
	}

	return nil
}

//Run 运行商户节点管理
func (c *Client) autoReconnectRemoteNode() error {

	var (
		err error
		//连接状态通道
		reconnect = make(chan bool, 1)
		//断开状态通道
		disconnected = make(chan struct{}, 1)
		//重连时的等待时间
		reconnectWait = 5
	)

	defer func() {
		close(reconnect)
		close(disconnected)
	}()

	//断开连接通知
	c.node.SetCloseHandler(func(n *owtp.OWTPNode, peer owtp.PeerInfo) {
		disconnected <- struct{}{}
	})

	//启动连接
	reconnect <- true

	//节点运行时
	for {
		select {
		case <-reconnect:
			//重新连接
			c.wm.Log.Info("Connecting to", c.config.remoteserver)
			err = c.connectRemoteNode()
			if err != nil {
				c.wm.Log.Errorf("Connect %s node failed unexpected error: %v", trustHostID, err)
				disconnected <- struct{}{}
			} else {
				c.wm.Log.Infof("Connect %s node successfully.", trustHostID)
			}

		case <-disconnected:
			//重新连接，前等待
			c.wm.Log.Info("Auto reconnect after", reconnectWait, "seconds...")
			time.Sleep(time.Duration(reconnectWait) * time.Second)
			reconnect <- true
		}
	}

	return nil
}

/*********** 客户服务平台业务方法调用 ***********/

func (c *Client) nodeDidConnectedServer() error {

	params := map[string]interface{}{
		"nodeInfo": TrustNodeInfo{
			NodeID:      c.node.NodeID(),
			NodeName:    nodeName,
			ConnectType: owtp.Websocket,
		},
	}

	err := c.node.Call(trustHostID, "newNodeJoin", params,
		true, func(resp owtp.Response) {
			if resp.Status != owtp.StatusSuccess {
				c.wm.Log.Error(resp.Msg)
			}
		})

	return err
}

//GetTransactionsByHeight
func (c *Client) GetTransactionsByHeight(height uint64) ([]*Transaction, error) {

	var (
		txs    []*Transaction
		retErr error
	)

	if !c.node.IsConnectPeer(trustHostID) {
		return nil, fmt.Errorf("client had disconnected: %s", trustHostID)
	}

	params := map[string]interface{}{
		"height": height,
	}

	err := c.node.Call(trustHostID, "getTransactionsByHeight", params,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				retErr = json.Unmarshal([]byte(resp.JsonData().Raw), &txs)
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return nil, err
	}

	return txs, retErr
}

//GetTransaction
func (c *Client) GetTransaction(txid string) (*Transaction, error) {

	var (
		tx     *Transaction
		retErr error
	)

	params := map[string]interface{}{
		"txid": txid,
	}

	err := c.node.Call(trustHostID, "getTransaction", params,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				retErr = json.Unmarshal([]byte(resp.JsonData().Raw), &tx)
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return nil, err
	}

	return tx, retErr
}

//CreateBatchAddress
func (c *Client) CreateBatchAddress(count, workerSize uint64) ([]string, error) {

	var (
		addrs  []string
		retErr error
	)

	params := map[string]interface{}{
		"count":      count,
		"workerSize": workerSize,
	}

	err := c.node.Call(trustHostID, "createBatchAddress", params,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				retErr = json.Unmarshal([]byte(resp.JsonData().Raw), &addrs)
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return nil, err
	}

	return addrs, retErr
}

//GetWalletBalance
func (c *Client) GetWalletBalance() (*openwallet.Balance, error) {

	var (
		walletBalance openwallet.Balance
		retErr        error
	)

	err := c.node.Call(trustHostID, "getWalletBalance", nil,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				balance := resp.JsonData().Get("balance")
				retErr = json.Unmarshal([]byte(balance.Raw), &walletBalance)
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return &walletBalance, err
	}

	return &walletBalance, retErr
}

//GetWalletAddress
func (c *Client) GetWalletAddress() ([]string, error) {

	var (
		addrs  []string
		retErr error
	)

	err := c.node.Call(trustHostID, "getWalletAddress", nil,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				retErr = json.Unmarshal([]byte(resp.JsonData().Raw), &addrs)
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return addrs, err
	}

	return addrs, retErr
}

//GetBlockByHeight
func (c *Client) GetBlockByHeight(height uint64) (*Block, error) {

	var (
		block  *Block
		retErr error
	)

	if !c.node.IsConnectPeer(trustHostID) {
		return nil, fmt.Errorf("client had disconnected: %s", trustHostID)
	}

	params := map[string]interface{}{
		"height": height,
	}

	err := c.node.Call(trustHostID, "getBlockByHeight", params,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				retErr = json.Unmarshal([]byte(resp.JsonData().Raw), &block)
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return nil, err
	}

	return block, retErr
}

//GetWalletStatus 获取钱包当前状态
func (c *Client) GetWalletStatus() (*WalletStatus, error) {

	var (
		WalletStatus *WalletStatus
		retErr       error
	)

	if !c.node.IsConnectPeer(trustHostID) {
		return nil, fmt.Errorf("client had disconnected: %s", trustHostID)
	}

	err := c.node.Call(trustHostID, "getWalletStatus", nil,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				retErr = json.Unmarshal([]byte(resp.JsonData().Raw), &WalletStatus)
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return nil, err
	}

	return WalletStatus, retErr

}

//打币
func (c *Client) TransFCoin(toAddress, toAmount string) (string, error) {

	var (
		txId   string
		retErr error
	)

	if !c.node.IsConnectPeer(trustHostID) {
		return "", fmt.Errorf("client had disconnected: %s", trustHostID)
	}
	params := map[string]interface{}{
		"toAddress": toAddress,
		"toAmount":  toAmount,
	}

	err := c.node.Call(trustHostID, "transFCoin", params,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				txId = resp.JsonData().Get("txId").String()
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return "", err
	}

	return txId, retErr
}

//打币
func (c *Client) SummaryToAddress(summaryAddress string) (string, string, string, error) {

	var (
		txId          string
		summaryAmount string
		feeAmount     string
		retErr        error
	)

	if !c.node.IsConnectPeer(trustHostID) {
		return "", "", "", fmt.Errorf("client had disconnected: %s", trustHostID)
	}
	params := map[string]interface{}{
		"summaryAddress": summaryAddress,
	}

	err := c.node.Call(trustHostID, "summaryToAddress", params,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				txId = resp.JsonData().Get("txId").String()
				summaryAmount = resp.JsonData().Get("summaryAmount").String()
				feeAmount = resp.JsonData().Get("feeAmount").String()
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return "", "", "", err
	}

	return txId, summaryAmount, feeAmount, retErr
}

//验证地址
func (c *Client) ValidateAddress(address string) (bool, error) {

	var (
		isValid bool
		retErr  error
	)

	if !c.node.IsConnectPeer(trustHostID) {
		return false, fmt.Errorf("client had disconnected: %s", trustHostID)
	}
	params := map[string]interface{}{
		"address": address,
	}

	err := c.node.Call(trustHostID, "validateAddress", params,
		true, func(resp owtp.Response) {
			if resp.Status == owtp.StatusSuccess {
				isValid = resp.JsonData().Get("isValid").Bool()
			} else {
				retErr = openwallet.Errorf(resp.Status, resp.Msg)
			}
		})
	if err != nil {
		return false, err
	}

	return isValid, retErr
}
