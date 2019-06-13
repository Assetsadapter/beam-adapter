package beam

import (
	"fmt"
	"github.com/blocktree/openwallet/common"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/shopspring/decimal"
	"math/big"
)

const (
	blockchainBucket = "blockchain" // blockchain dataset
	//periodOfTask      = 5 * time.Second // task interval
	maxExtractingSize = 10 // thread count
)

//BEAMBlockScanner BEAM block scanner
type BEAMBlockScanner struct {
	*openwallet.BlockScannerBase

	CurrentBlockHeight   uint64         //当前区块高度
	extractingCH         chan struct{}  //扫描工作令牌
	wm                   *WalletManager //钱包管理者
	RescanLastBlockCount uint64         //重扫上N个区块数量
}

//ExtractResult 扫描完成的提取结果
type ExtractResult struct {
	extractData map[string][]*openwallet.TxExtractData
	TxID        string
	BlockHeight uint64
	Success     bool
}

//SaveResult result
type SaveResult struct {
	TxID        string
	BlockHeight uint64
	Success     bool
}

// NewEOSBlockScanner create a block scanner
func NewBEAMBlockScanner(wm *WalletManager) *BEAMBlockScanner {
	bs := BEAMBlockScanner{
		BlockScannerBase: openwallet.NewBlockScannerBase(),
	}

	bs.extractingCH = make(chan struct{}, maxExtractingSize)
	bs.wm = wm

	bs.RescanLastBlockCount = 0

	// set task
	bs.SetTask(bs.ScanBlockTask)

	return &bs
}

//GetBalanceByAddress 查询地址余额
func (bs *BEAMBlockScanner) GetBalanceByAddress(address ...string) ([]*openwallet.Balance, error) {
	wallet, err := bs.wm.walletClient.GetWalletStatus()
	if err != nil {
		return nil, err
	}

	confirmBalance := common.IntToDecimals(int64(wallet.Available), bs.wm.Decimal())
	unconfirmedBalance := common.IntToDecimals(int64(wallet.Receiving), bs.wm.Decimal())
	balance := confirmBalance.Add(unconfirmedBalance)

	b := &openwallet.Balance{
		Symbol:           bs.wm.Symbol(),
		Balance:          balance.String(),
		ConfirmBalance:   confirmBalance.String(),
		UnconfirmBalance: unconfirmedBalance.String(),
	}

	return []*openwallet.Balance{b}, nil
}

//
//GetCurrentBlock 获取当前最新区块
func (bs *BEAMBlockScanner) GetCurrentBlock() (*Block, error) {

	wallet, err := bs.wm.walletClient.GetWalletStatus()
	if err != nil {
		return nil, err
	}

	block := &Block{
		Height:        wallet.CurrentHeight,
		Hash:          wallet.CurrentStateHash,
		PrevBlockHash: wallet.PrevStateHash,
	}

	return block, nil
}

//GetBlockHeight 获取区块链高度
func (bs *BEAMBlockScanner) GetBlockHeight() (uint64, error) {

	block, err := bs.GetCurrentBlock()
	if err != nil {
		return 0, err
	}
	return block.Height, nil
}

//GetCurrentBlockHeader 获取当前区块高度
func (bs *BEAMBlockScanner) GetCurrentBlockHeader() (*openwallet.BlockHeader, error) {

	block, err := bs.GetCurrentBlock()
	if err != nil {
		return nil, err
	}

	return &openwallet.BlockHeader{Height: block.Height, Hash: block.Hash}, nil
}

//SetRescanBlockHeight 重置区块链扫描高度
func (bs *BEAMBlockScanner) SetRescanBlockHeight(height uint64) error {
	height = height - 1
	if height < 0 {
		return fmt.Errorf("block height to rescan must greater than 0.")
	}
	block, err := bs.GetBlockByHeight(height)
	if err != nil {
		return err
	}

	bs.wm.SaveLocalNewBlock(height, block.Hash)

	return nil
}

func (bs *BEAMBlockScanner) GetBlockByHash(hash string) (*Block, error) {
	return bs.wm.walletClient.GetBlockByHash(hash)
}

func (bs *BEAMBlockScanner) GetBlockByHeight(height uint64) (*Block, error) {
	return bs.wm.walletClient.GetBlockByHeight(height)
}

//GetScannedBlockHeader 获取当前扫描的区块头
func (bs *BEAMBlockScanner) GetScannedBlockHeader() (*openwallet.BlockHeader, error) {

	var (
		blockHeader *openwallet.BlockHeader
		blockHeight uint64 = 0
		hash        string
		err         error
	)

	blockHeight, hash = bs.wm.GetLocalNewBlock()

	//如果本地没有记录，查询接口的高度
	if blockHeight == 0 {
		blockHeader, err = bs.GetCurrentBlockHeader()
		if err != nil {

			return nil, err
		}
		blockHeight = blockHeader.Height
		//就上一个区块链为当前区块
		blockHeight = blockHeight - 1

		block, err := bs.GetBlockByHeight(blockHeight)
		if err != nil {
			return nil, err
		}
		hash = block.Hash
	}

	return &openwallet.BlockHeader{Height: blockHeight, Hash: hash}, nil
}

//GetScannedBlockHeight 获取已扫区块高度
func (bs *BEAMBlockScanner) GetScannedBlockHeight() uint64 {
	localHeight, _ := bs.wm.GetLocalNewBlock()
	return localHeight
}

//GetGlobalMaxBlockHeight 获取区块链全网最大高度
func (bs *BEAMBlockScanner) GetGlobalMaxBlockHeight() uint64 {

	height, err := bs.GetBlockHeight()
	if err != nil {
		return 0
	}

	return height
}

//GetTransaction
func (bs *BEAMBlockScanner) GetTransaction(hash string) (*Transaction, error) {
	return bs.wm.walletClient.GetTransaction(hash)
}

//ScanBlockTask 扫描任务
func (bs *BEAMBlockScanner) ScanBlockTask() {

	//获取本地区块高度
	blockHeader, err := bs.GetScannedBlockHeader()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block height; unexpected error: %v", err)
		return
	}

	currentHeight := blockHeader.Height
	currentHash := blockHeader.Hash

	for {

		if !bs.Scanning {
			//区块扫描器已暂停，马上结束本次任务
			return
		}

		//获取最大高度
		maxHeight, err := bs.GetBlockHeight()
		if err != nil {
			//下一个高度找不到会报异常
			bs.wm.Log.Std.Info("block scanner can not get rpc-server block height; unexpected error: %v", err)
			break
		}

		//是否已到最新高度
		if currentHeight >= maxHeight {
			bs.wm.Log.Std.Info("block scanner has scanned full chain data. Current height: %d", maxHeight)
			break
		}

		//继续扫描下一个区块
		currentHeight = currentHeight + 1

		bs.wm.Log.Std.Info("block scanner scanning height: %d ...", currentHeight)

		block, err := bs.GetBlockByHeight(currentHeight)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

			//记录未扫区块
			unscanRecord := NewUnscanRecord(currentHeight, "", err.Error())
			bs.SaveUnscanRecord(unscanRecord)
			bs.wm.Log.Std.Info("block height: %d extract failed.", currentHeight)
			continue
		}

		isFork := false

		//判断hash是否上一区块的hash
		if currentHash != block.PrevBlockHash {

			bs.wm.Log.Std.Info("block has been fork on height: %d.", currentHeight)
			bs.wm.Log.Std.Info("block height: %d local hash = %s ", currentHeight-1, currentHash)
			bs.wm.Log.Std.Info("block height: %d mainnet hash = %s ", currentHeight-1, block.PrevBlockHash)

			bs.wm.Log.Std.Info("delete recharge records on block height: %d.", currentHeight-1)

			//查询本地分叉的区块
			forkBlock, _ := bs.wm.GetLocalBlock(currentHeight - 1)

			//删除上一区块链的所有充值记录
			//bs.DeleteRechargesByHeight(currentHeight - 1)
			//删除上一区块链的未扫记录
			bs.wm.DeleteUnscanRecord(currentHeight - 1)
			currentHeight = currentHeight - 2 //倒退2个区块重新扫描
			if currentHeight <= 0 {
				currentHeight = 1
			}

			localBlock, err := bs.wm.GetLocalBlock(currentHeight)
			if err != nil {
				bs.wm.Log.Std.Error("block scanner can not get local block; unexpected error: %v", err)

				//查找core钱包的RPC
				bs.wm.Log.Info("block scanner prev block height:", currentHeight)

				localBlock, err = bs.GetBlockByHeight(currentHeight)
				if err != nil {
					bs.wm.Log.Std.Error("block scanner can not get prev block; unexpected error: %v", err)
					break
				}

			}

			//重置当前区块的hash
			currentHash = localBlock.Hash

			bs.wm.Log.Std.Info("rescan block on height: %d, hash: %s .", currentHeight, currentHash)

			//重新记录一个新扫描起点
			bs.wm.SaveLocalNewBlock(localBlock.Height, localBlock.Hash)

			isFork = true

			if forkBlock != nil {

				//通知分叉区块给观测者，异步处理
				bs.newBlockNotify(forkBlock, isFork)
			}

		} else {

			err = bs.BatchExtractTransaction(block.Height, block.Hash)
			if err != nil {
				bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
				return
			}

			//重置当前区块的hash
			currentHash = block.Hash

			//保存本地新高度
			bs.wm.SaveLocalNewBlock(currentHeight, currentHash)
			bs.wm.SaveLocalBlock(block)

			isFork = false

			//通知新区块给观测者，异步处理
			bs.newBlockNotify(block, isFork)
		}

	}

	//重扫前N个块，为保证记录找到
	for i := currentHeight - bs.RescanLastBlockCount; i < currentHeight; i++ {
		bs.scanBlock(i)
	}

	//重扫失败区块
	bs.RescanFailedRecord()

}

//ScanBlock 扫描指定高度区块
func (bs *BEAMBlockScanner) ScanBlock(height uint64) error {

	block, err := bs.scanBlock(height)
	if err != nil {
		return err
	}

	//通知新区块给观测者，异步处理
	bs.newBlockNotify(block, false)

	return nil
}

func (bs *BEAMBlockScanner) scanBlock(height uint64) (*Block, error) {

	block, err := bs.GetBlockByHeight(height)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

		//记录未扫区块
		unscanRecord := NewUnscanRecord(height, "", err.Error())
		bs.SaveUnscanRecord(unscanRecord)
		bs.wm.Log.Std.Info("block height: %d extract failed.", height)
		return nil, err
	}

	bs.wm.Log.Std.Info("block scanner scanning height: %d ...", block.Height)

	err = bs.BatchExtractTransaction(block.Height, block.Hash)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
	}

	return block, nil
}

//rescanFailedRecord 重扫失败记录
func (bs *BEAMBlockScanner) RescanFailedRecord() {

	var (
		blockMap = make(map[uint64][]string)
	)

	list, err := bs.wm.GetUnscanRecords()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get rescan data; unexpected error: %v", err)
	}

	//组合成批处理
	for _, r := range list {

		if _, exist := blockMap[r.BlockHeight]; !exist {
			blockMap[r.BlockHeight] = make([]string, 0)
		}

		if len(r.TxID) > 0 {
			arr := blockMap[r.BlockHeight]
			arr = append(arr, r.TxID)

			blockMap[r.BlockHeight] = arr
		}
	}

	for height, _ := range blockMap {

		if height == 0 {
			continue
		}

		bs.wm.Log.Std.Info("block scanner rescanning height: %d ...", height)

		block, err := bs.GetBlockByHeight(height)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)
			continue
		}

		err = bs.BatchExtractTransaction(height, block.Hash)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
			continue
		}

		//删除未扫记录
		bs.wm.DeleteUnscanRecord(height)
	}

	//删除未没有找到交易记录的重扫记录
	bs.wm.DeleteUnscanRecordNotFindTX()
}

//newBlockNotify 获得新区块后，通知给观测者
func (bs *BEAMBlockScanner) newBlockNotify(block *Block, isFork bool) {
	header := block.BlockHeader(bs.wm.Symbol())
	header.Fork = isFork
	bs.NewBlockNotify(header)
}

//BatchExtractTransaction 批量提取交易单
//bitcoin 1M的区块链可以容纳3000笔交易，批量多线程处理，速度更快
func (bs *BEAMBlockScanner) BatchExtractTransaction(blockHeight uint64, blockHash string) error {

	var (
		quit       = make(chan struct{})
		done       = 0 //完成标记
		failed     = 0
		shouldDone = 0 //需要完成的总数
	)

	// 获取查找本地交易单和远程服务上的交易单
	txs, err := bs.wm.GetTransactionsByHeight(blockHeight)
	if err != nil {
		return err
	}

	if len(txs) == 0 {
		return nil
	}

	shouldDone = len(txs) //需要完成的总数

	//生产通道
	producer := make(chan ExtractResult)
	defer close(producer)

	//消费通道
	worker := make(chan ExtractResult)
	defer close(worker)

	//保存工作
	saveWork := func(height uint64, result chan ExtractResult) {
		//回收创建的地址
		for gets := range result {

			if gets.Success {

				notifyErr := bs.newExtractDataNotify(height, gets.extractData)
				//saveErr := bs.SaveRechargeToWalletDB(height, gets.Recharges)
				if notifyErr != nil {
					failed++ //标记保存失败数
					bs.wm.Log.Std.Info("newExtractDataNotify unexpected error: %v", notifyErr)
				}

			} else {
				//记录未扫区块
				unscanRecord := NewUnscanRecord(height, "", "")
				bs.SaveUnscanRecord(unscanRecord)
				bs.wm.Log.Std.Info("block height: %d extract failed.", height)
				failed++ //标记保存失败数
			}
			//累计完成的线程数
			done++
			if done == shouldDone {
				//bs.wm.Log.Std.Info("done = %d, shouldDone = %d ", done, len(txs))
				close(quit) //关闭通道，等于给通道传入nil
			}
		}
	}

	//提取工作
	extractWork := func(eblockHeight uint64, eBlockHash string, mTxs []*Transaction, eProducer chan ExtractResult) {
		for _, tx := range mTxs {
			bs.extractingCH <- struct{}{}
			//shouldDone++
			go func(mBlockHeight uint64, mTx *Transaction, end chan struct{}, mProducer chan<- ExtractResult) {

				//导出提出的交易
				mProducer <- bs.ExtractTransaction(mBlockHeight, eBlockHash, mTx, bs.ScanTargetFunc)
				//释放
				<-end

			}(eblockHeight, tx, bs.extractingCH, eProducer)
		}
	}

	/*	开启导出的线程	*/

	//独立线程运行消费
	go saveWork(blockHeight, worker)

	//独立线程运行生产
	go extractWork(blockHeight, blockHash, txs, producer)

	//以下使用生产消费模式
	bs.extractRuntime(producer, worker, quit)

	if failed > 0 {
		return fmt.Errorf("block scanner saveWork failed")
	} else {
		return nil
	}

	//return nil
}

//extractRuntime 提取运行时
func (bs *BEAMBlockScanner) extractRuntime(producer chan ExtractResult, worker chan ExtractResult, quit chan struct{}) {

	var (
		values = make([]ExtractResult, 0)
	)

	for {

		var activeWorker chan<- ExtractResult
		var activeValue ExtractResult

		//当数据队列有数据时，释放顶部，传输给消费者
		if len(values) > 0 {
			activeWorker = worker
			activeValue = values[0]

		}

		select {

		//生成者不断生成数据，插入到数据队列尾部
		case pa := <-producer:
			values = append(values, pa)
		case <-quit:
			//退出
			//bs.wm.Log.Std.Info("block scanner have been scanned!")
			return
		case activeWorker <- activeValue:
			//wm.Log.Std.Info("Get %d", len(activeValue))
			values = values[1:]
		}
	}

}

//提取交易单
func (bs *BEAMBlockScanner) ExtractTransaction(blockHeight uint64, blockHash string, trx *Transaction, scanTargetFunc openwallet.BlockScanTargetFunc) ExtractResult {
	var (
		success = true
		result  = ExtractResult{
			BlockHeight: blockHeight,
			TxID:        trx.TxID,
			extractData: make(map[string][]*openwallet.TxExtractData),
		}
	)

	if len(blockHash) == 0 {
		block, err := bs.GetBlockByHeight(trx.BlockHeight)
		if err == nil {
			blockHash = block.Hash
			blockHeight = block.Height
		}
	}

	trx.BlockHeight = blockHeight
	trx.BlockHash = blockHash

	//提出交易单明细
	from := trx.Sender
	to := trx.Receiver

	//bs.wm.Log.Std.Info("block scanner scanning tx: %+v", txid)
	//订阅地址为交易单中的发送者
	accountId, ok1 := scanTargetFunc(openwallet.ScanTarget{
		Address:          from,
		BalanceModelType: openwallet.BalanceModelTypeAddress,
	})
	//订阅地址为交易单中的接收者
	accountId2, ok2 := scanTargetFunc(openwallet.ScanTarget{
		Address:          to,
		BalanceModelType: openwallet.BalanceModelTypeAddress,
	})

	//相同账户
	if accountId == accountId2 && len(accountId) > 0 && len(accountId2) > 0 {
		bs.InitExtractResult(trx, accountId, &result, 0)
	} else {
		if ok1 {
			bs.InitExtractResult(trx, accountId, &result, 1)
		}

		if ok2 {
			bs.InitExtractResult(trx, accountId2, &result, 2)
		}
	}

	success = true

	result.Success = success
	return result

}

//InitTronExtractResult operate = 0: 输入输出提取，1: 输入提取，2：输出提取
func (bs *BEAMBlockScanner) InitExtractResult(tx *Transaction, sourceKey string, result *ExtractResult, operate int64) {

	txExtractDataArray := result.extractData[sourceKey]
	if txExtractDataArray == nil {
		txExtractDataArray = make([]*openwallet.TxExtractData, 0)
	}

	txExtractData := &openwallet.TxExtractData{}

	status := "1"
	reason := ""

	amount := decimal.Zero
	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: false,
	}
	value := new(big.Int)
	value.SetUint64(tx.Value)
	amount = common.BigIntToDecimals(value, bs.wm.Decimal())

	transx := &openwallet.Transaction{
		Fees:        "0",
		Coin:        coin,
		BlockHash:   tx.BlockHash,
		BlockHeight: tx.BlockHeight,
		TxID:        tx.TxID,
		Decimal:     bs.wm.Decimal(),
		Amount:      amount.String(),
		ConfirmTime: tx.CreateTime,
		From:        []string{tx.Sender + ":" + amount.String()},
		To:          []string{tx.Receiver + ":" + amount.String()},
		Status:      status,
		Reason:      reason,
	}

	wxID := openwallet.GenTransactionWxID(transx)
	transx.WxID = wxID

	txExtractData.Transaction = transx
	if operate == 0 {
		bs.extractTxInput(tx, txExtractData)
		bs.extractTxOutput(tx, txExtractData)
	} else if operate == 1 {
		bs.extractTxInput(tx, txExtractData)
	} else if operate == 2 {
		bs.extractTxOutput(tx, txExtractData)
	}

	txExtractDataArray = append(txExtractDataArray, txExtractData)
	result.extractData[sourceKey] = txExtractDataArray
}

//extractTxInput 提取交易单输入部分,无需手续费，所以只包含1个TxInput
func (bs *BEAMBlockScanner) extractTxInput(tx *Transaction, txExtractData *openwallet.TxExtractData) {

	amount := decimal.Zero
	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: false,
	}

	value := new(big.Int)
	value.SetUint64(tx.Value)
	amount = common.BigIntToDecimals(value, bs.wm.Decimal())

	//主网from交易转账信息，第一个TxInput
	txInput := &openwallet.TxInput{}
	txInput.Recharge.Sid = openwallet.GenTxInputSID(tx.TxID, bs.wm.Symbol(), coin.ContractID, uint64(0))
	txInput.Recharge.TxID = tx.TxID
	txInput.Recharge.Address = tx.Sender
	txInput.Recharge.Coin = coin
	txInput.Recharge.Amount = amount.String()
	txInput.Recharge.BlockHash = tx.BlockHash
	txInput.Recharge.BlockHeight = tx.BlockHeight
	txInput.Recharge.Index = 0 //账户模型填0
	txInput.Recharge.CreateAt = tx.CreateTime
	txExtractData.TxInputs = append(txExtractData.TxInputs, txInput)

	//手续费也作为一个输出s
	fee := new(big.Int)
	fee.SetUint64(tx.Fee)
	fees := common.BigIntToDecimals(fee, bs.wm.Decimal())
	tmp := *txInput
	feeCharge := &tmp
	feeCharge.Amount = fees.String()
	txExtractData.TxInputs = append(txExtractData.TxInputs, feeCharge)
}

//extractTxOutput 提取交易单输入部分,只有一个TxOutPut
func (bs *BEAMBlockScanner) extractTxOutput(tx *Transaction, txExtractData *openwallet.TxExtractData) {

	amount := decimal.Zero
	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: false,
	}

	value := new(big.Int)
	value.SetUint64(tx.Value)
	amount = common.BigIntToDecimals(value, bs.wm.Decimal())

	//主网to交易转账信息,只有一个TxOutPut
	txOutput := &openwallet.TxOutPut{}
	txOutput.Recharge.Sid = openwallet.GenTxOutPutSID(tx.TxID, bs.wm.Symbol(), coin.ContractID, uint64(0))
	txOutput.Recharge.TxID = tx.TxID
	txOutput.Recharge.Address = tx.Receiver
	txOutput.Recharge.Coin = coin
	txOutput.Recharge.Amount = amount.String()
	txOutput.Recharge.BlockHash = tx.BlockHash
	txOutput.Recharge.BlockHeight = tx.BlockHeight
	txOutput.Recharge.Index = 0 //账户模型填0
	txOutput.Recharge.CreateAt = tx.CreateTime
	txExtractData.TxOutputs = append(txExtractData.TxOutputs, txOutput)
}

//newExtractDataNotify 发送通知
//发送通知
func (bs *BEAMBlockScanner) newExtractDataNotify(height uint64, extractData map[string][]*openwallet.TxExtractData) error {
	for o, _ := range bs.Observers {
		for key, array := range extractData {
			for _, data := range array {
				err := o.BlockExtractDataNotify(key, data)
				if err != nil {
					bs.wm.Log.Error("BlockExtractDataNotify unexpected error:", err)
					//记录未扫区块
					unscanRecord := NewUnscanRecord(height, "", "ExtractData Notify failed.")
					err = bs.SaveUnscanRecord(unscanRecord)
					if err != nil {
						bs.wm.Log.Std.Error("block height: %d, save unscan record failed. unexpected error: %v", height, err.Error())
					}
				}
			}
		}
	}
	return nil
}

//ExtractTransactionData
func (bs *BEAMBlockScanner) ExtractTransactionData(txid string, scanAddressFunc openwallet.BlockScanTargetFunc) (map[string][]*openwallet.TxExtractData, error) {
	tx, err := bs.wm.GetTransaction(txid)
	if err != nil {
		return nil, err
	}
	result := bs.ExtractTransaction(0, "", tx, scanAddressFunc)
	return result.extractData, nil
}
