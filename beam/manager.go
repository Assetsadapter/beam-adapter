package beam

import (
	"fmt"
	"github.com/blocktree/openwallet/common"
	"github.com/blocktree/openwallet/common/file"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/blocktree/openwallet/owtp"
	"github.com/blocktree/openwallet/timer"
	"github.com/shopspring/decimal"
	"math/big"
	"time"
)

type WalletManager struct {
	openwallet.AssetsAdapterBase

	node            *owtp.OWTPNode
	Config          *WalletConfig                   // 节点配置
	Decoder         openwallet.AddressDecoder       //地址编码器
	TxDecoder       openwallet.TransactionDecoder   //交易单编码器
	Log             *log.OWLogger                   //日志工具
	ContractDecoder openwallet.SmartContractDecoder //智能合约解析器
	Blockscanner    *BEAMBlockScanner               //区块扫描器
	walletClient    *WalletClient                   //本地封装的http client
	client          *Client                         //节点作为客户端
	server          *Server                         //节点作为服务端
}

func NewWalletManager() *WalletManager {
	wm := WalletManager{}
	wm.Config = NewConfig(Symbol)
	wm.Blockscanner = NewBEAMBlockScanner(&wm)
	//wm.Decoder = NewAddressDecoder(&wm)
	wm.TxDecoder = NewTransactionDecoder(&wm)
	wm.Log = log.NewOWLogger(wm.Symbol())
	return &wm
}

func (wm WalletManager) CreateRemoteWalletAddress(count, workerSize uint64) ([]string, error) {
	if wm.Config.enableserver {
		return nil, fmt.Errorf("server mode can not create remote address, use create local address")
	}

	return wm.client.CreateBatchAddress(count, workerSize)
}

func (wm WalletManager) GetRemoteWalletAddress() ([]string, error) {
	if wm.Config.enableserver {
		return nil, fmt.Errorf("server mode can not create remote address, use create local address")
	}

	return wm.client.GetWalletAddress()
}

func (wm WalletManager) GetRemoteWalletBalance() (*openwallet.Balance, error) {

	if wm.Config.enableserver {
		return nil, fmt.Errorf("server mode can not get remote wallet balance, use get wallet balance")
	}

	b, err := wm.client.GetWalletBalance()
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (wm WalletManager) CreateLocalWalletAddress(count, workerSize uint64) ([]string, error) {
	return wm.walletClient.CreateBatchAddress(count, workerSize)
}

func (wm WalletManager) GetLocalWalletBalance() (*openwallet.Balance, error) {

	b, err := wm.Blockscanner.GetBalanceByAddress()
	if err != nil {
		return nil, err
	}

	if len(b) == 0 {
		return nil, fmt.Errorf("can not get wallet balance")
	}
	return b[0], nil
}

func (wm WalletManager) GetLocalWalletAddress() ([]string, error) {
	return wm.walletClient.GetAddressList()
}

//GetTransactionsByHeight
func (wm *WalletManager) GetTransaction(txid string) (*Transaction, error) {

	localTx, err := wm.walletClient.GetTransaction(txid)
	if err != nil {
		wm.Log.Errorf("Local GetTransaction failed, unexpected error %v", err)
	}

	if localTx != nil {
		return localTx, nil
	}

	if wm.client != nil {
		remoteTx, err := wm.client.GetTransaction(txid)
		if err != nil {
			wm.Log.Errorf("Remote GetTransactionsByHeight failed, unexpected error %v", err)
		}

		if remoteTx != nil {
			return remoteTx, nil
		}
	}

	return nil, fmt.Errorf("can not find transaction")
}

//GetTransactionsByHeight
func (wm *WalletManager) GetTransactionsByHeight(height uint64) ([]*Transaction, error) {

	trxMap := make(map[string]*Transaction, 0)
	trxs := make([]*Transaction, 0)

	localTrxs, err := wm.walletClient.GetTransactionsByHeight(height)
	if err != nil {
		wm.Log.Errorf("Local GetTransactionsByHeight failed, unexpected error %v", err)
		return nil, err
	}

	for _, tx := range localTrxs {
		trxMap[tx.TxID] = tx
	}

	if wm.client != nil {
		remoteTrxs, err := wm.client.GetTransactionsByHeight(height)
		if err != nil {
			wm.Log.Errorf("Remote GetTransactionsByHeight failed, unexpected error %v", err)
			return nil, err
		}

		for _, tx := range remoteTrxs {
			trxMap[tx.TxID] = tx
		}

	}

	for _, tx := range trxMap {
		trxs = append(trxs, tx)
	}

	return trxs, nil
}

func (wm WalletManager) GetRemoteBlockByHeight(height uint64) (*Block, error) {
	//if wm.Config.enableserver {
	//	return nil, fmt.Errorf("server mode can not create remote address, use create local address")
	//}

	return wm.client.GetBlockByHeight(height)
}

func (wm *WalletManager) StartSummaryWallet() error {

	var (
		endRunning = make(chan bool, 1)
	)

	cycleTime := wm.Config.summaryperiod
	if len(cycleTime) == 0 {
		cycleTime = "1m"
	}

	cycleSec, err := time.ParseDuration(cycleTime)
	if err != nil {
		return err
	}

	if len(wm.Config.summaryaddress) == 0 {
		return fmt.Errorf("summary address is not setup")
	}

	if len(wm.Config.summarythreshold) == 0 {
		return fmt.Errorf("summary threshold is not setup")
	}

	wm.Log.Infof("The timer for summary task start now. Execute by every %v seconds.", cycleSec.Seconds())

	//启动钱包汇总程序
	sumTimer := timer.NewTask(cycleSec, wm.SummaryWallets)
	sumTimer.Start()

	//马上执行一次
	wm.SummaryWallets()

	<-endRunning

	return nil
}

//SummaryWallets 执行汇总流程
func (wm *WalletManager) SummaryWallets() {

	wm.Log.Infof("[Summary Task Start]------%s", common.TimeFormat("2006-01-02 15:04:05"))

	txId, _, _, err := wm.SummaryWalletProcess(wm.Config.summaryaddress)
	if err != nil {
		wm.Log.Errorf("summary wallet unexpected error: %v", err)
	}

	wm.Log.Infof("[Summary Task End] txId =%s ------%s", txId, common.TimeFormat("2006-01-02 15:04:05"))

	//:清楚超时的交易
	wm.ClearExpireTx()
}

//汇总到目标地址
//return txId,summaryAmount,feeAmount,err
func (wm *WalletManager) SummaryWalletProcess(summaryToAddress string) (string, string, string, error) {
	if summaryToAddress == "" {
		return "", "", "", fmt.Errorf("param summaryToAddress is null")

	}
	status, err := wm.walletClient.GetWalletStatus()
	if err != nil {
		return "", "", "", fmt.Errorf("get local wallet balance failed, unexpected error: %v", err)
	}

	balance := common.IntToDecimals(int64(status.Available), wm.Decimal())
	threshold, _ := decimal.NewFromString(wm.Config.summarythreshold)

	wm.Log.Infof("Summary Wallet Current Balance: %v, threshold: %v", balance.String(), threshold.String())

	//如果余额大于阀值，汇总的地址
	if balance.GreaterThan(threshold) {

		feesDec, _ := decimal.NewFromString(wm.Config.fixfees)
		sumAmount := balance.Sub(feesDec)

		wm.Log.Infof("Summary Wallet Current Balance = %s ", balance.String())
		wm.Log.Infof("Summary Wallet Summary Amount = %s ", sumAmount.String())
		wm.Log.Infof("Summary Wallet Summary Fee = %s ", wm.Config.fixfees)
		wm.Log.Infof("Summary Wallet Summary Address = %v ", wm.Config.summaryaddress)
		wm.Log.Infof("Summary Wallet Start Create Summary Transaction")

		fixFees := common.StringNumToBigIntWithExp(wm.Config.fixfees, wm.Decimal())

		//检查余额是否超过最低转账
		addrBalance_BI := new(big.Int)
		addrBalance_BI.SetUint64(status.Available)
		sumAmount_BI := new(big.Int)
		//减去手续费
		sumAmount_BI.Sub(addrBalance_BI, fixFees)
		if sumAmount_BI.Cmp(big.NewInt(0)) <= 0 {
			return "", "", "", fmt.Errorf("summary amount not enough pay fee, ")
		}

		//取一个地址作为发送
		addresses, err := wm.walletClient.GetAddressList()
		if err != nil {
			return "", "", "", err
		}

		if addresses == nil || len(addresses) == 0 {
			return "", "", "", fmt.Errorf("wallet address is not created")
		}

		from := addresses[0]

		txid, err := wm.walletClient.SendTransaction(from, summaryToAddress, sumAmount_BI.Uint64(), fixFees.Uint64(), "")
		if err != nil {
			return "", "", "", err
		}

		wm.Log.Infof("[Success] txid: %s", txid)
		fee_dec := decimal.NewFromBigInt(fixFees, wm.Decimal()*-1)
		summary_dec := decimal.NewFromBigInt(sumAmount_BI, wm.Decimal()*-1).Add(feesDec).Neg()
		return txid, summary_dec.String(), fee_dec.String(), nil
		////完成一次汇总备份一次wallet.db
		//backErr := wm.BackupWalletData()
		//if backErr != nil {
		//	wm.Log.Infof("Backup wallet data failed: %v", backErr)
		//} else {
		//	wm.Log.Infof("Backup wallet data success")
		//}

	}

	return "", "", "", nil
}

//ClearExpireTx
func (wm *WalletManager) ClearExpireTx() error {

	txs, err := wm.walletClient.GetTransactionsByStatus(TxStatusInProgress)
	if err != nil {
		return err
	}

	currentServerTime := time.Now()

	for _, tx := range txs {
		//计算交易发送过期时间
		txCreateTimestamp := time.Unix(tx.CreateTime, 0)
		expiredTime := txCreateTimestamp.Add(wm.Config.txsendingtimeout)

		//log.Infof("txCreateTimestamp = %s", txCreateTimestamp.String())
		//log.Infof("currentServerTime = %s", currentServerTime.String())
		//log.Infof("expiredTime = %s", expiredTime.String())

		if currentServerTime.Unix() > expiredTime.Unix() {

			log.Infof("In Progress Tx: %s is expired", tx.TxID)

			flag, cancelErr := wm.walletClient.CancelTx(tx.TxID)
			if cancelErr != nil {
				return cancelErr
			}
			log.Infof("Cancel Tx: %s = %v", tx.TxID, flag)
		}
	}
	return nil
}

//BackupWalletData
func (wm *WalletManager) BackupWalletData() error {

	//备份钱包文件
	return file.Copy(wm.Config.walletdatafile, wm.Config.walletdatabackupdir)

}
