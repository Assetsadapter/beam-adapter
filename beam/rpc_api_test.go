package beam

import (
	"github.com/blocktree/openwallet/log"
	"testing"
)

func TestWalletClient_GetBlockchainInfo(t *testing.T) {
	b, err := tw.walletClient.GetBlockchainInfo()
	if err != nil {
		t.Errorf("GetBlockchainInfo failed unexpected error: %v\n", err)
	} else {
		log.Infof("GetBlockchainInfo info: %+v\n", b)
	}
}

func TestWalletClient_CreateAddress(t *testing.T) {
	addr, err := tw.walletClient.CreateAddress()
	if err != nil {
		t.Errorf("CreateAddress failed unexpected error: %v\n", err)
	} else {
		log.Infof("CreateAddress info: %s", addr)
	}
}

func TestWalletClient_CreateBatchAddress(t *testing.T) {
	addrs, err := tw.walletClient.CreateBatchAddress(2000, 20)
	if err != nil {
		t.Errorf("CreateBatchAddress failed unexpected error: %v\n", err)
		return
	}

	for i, a := range addrs {
		log.Infof("%d: %s", i, a)
	}
}

func TestWalletClient_GetBlockByHeight(t *testing.T) {

	block, err := tw.walletClient.GetBlockByHeight(161025)
	if err != nil {
		t.Errorf("GetBlockByHeight failed unexpected error: %v\n", err)
	} else {
		log.Infof("GetBlockByHeight info: %+v", block)
	}
}

func TestWalletClient_GetBlockByHash(t *testing.T) {

	block, err := tw.walletClient.GetBlockByHash("c9b4584d7a8eda016c26b4c8cb6f55775c415eaf36c460b9180df00f0cd3bbf3")
	if err != nil {
		t.Errorf("GetBlockByHash failed unexpected error: %v\n", err)
	} else {
		log.Infof("GetBlockByHash info: %+v", block)
	}
}

func TestWalletClient_GetBlockByKernel(t *testing.T) {

	block, err := tw.walletClient.GetBlockByKernel("22abe54b476951179f58ff8da9f06332fc138e9f33f35c3f04b7ea3c71d45fd6")
	if err != nil {
		t.Errorf("GetBlockByKernel failed unexpected error: %v\n", err)
	} else {
		log.Infof("GetBlockByKernel info: %+v", block)
	}
}

func TestWalletClient_GetTransaction(t *testing.T) {
	tx, err := tw.walletClient.GetTransaction("72f8f349f9244b11b0e6471250ca68a1")
	if err != nil {
		t.Errorf("GetTransaction failed unexpected error: %v\n", err)
	} else {
		log.Infof("GetTransaction info: %+v", tx)
	}
}

func TestWalletClient_GetTransactionsByHeight(t *testing.T) {
	txs, err := tw.walletClient.GetTransactionsByHeight(237304)
	if err != nil {
		t.Errorf("GetTransactionsByHeight failed unexpected error: %v\n", err)
		return
	}

	for i, tx := range txs {
		log.Infof("%d: %+v", i, tx)
	}
}

func TestWalletClient_GetAddressList(t *testing.T) {
	addrs, err := tw.walletClient.GetAddressList()
	if err != nil {
		t.Errorf("GetAddressList failed unexpected error: %v\n", err)
		return
	}

	for i, a := range addrs {
		log.Infof("%d: %s", i, a)
	}
}

func TestWalletClient_GetWalletStatus(t *testing.T) {
	wallet, err := tw.walletClient.GetWalletStatus()
	if err != nil {
		t.Errorf("GetWalletStatus failed unexpected error: %v\n", err)
		return
	}

	log.Infof("wallet: %+v", wallet)
}

func TestWalletClient_SendTransaction(t *testing.T) {
	from := "21aff5eb4da2591321ac12bb280ac69ea39a33472166c600ec122cf3381b6c9e772"
	to := "19179fae58832b5a59129cd866905646d7547d1dddd1f97c3663affb924a01fa65c"
	amount := uint64(45738)
	fee := uint64(1)
	txid, err := tw.walletClient.SendTransaction(from, to, amount, fee, "")
	if err != nil {
		t.Errorf("GetWalletStatus failed unexpected error: %v\n", err)
		return
	}

	log.Infof("txid: %s", txid)
}

func TestWalletClient_GetTransactionsByStatus(t *testing.T) {
	txs, err := tw.walletClient.GetTransactionsByStatus(TxStatusInProgress)
	if err != nil {
		t.Errorf("GetTransactionsByStatus failed unexpected error: %v\n", err)
		return
	}

	for i, tx := range txs {
		log.Infof("%d: %+v", i, tx)
	}

}

func TestWalletClient_CancelTx(t *testing.T) {
	flag, err := tw.walletClient.CancelTx("46bf4426eb8142f58898ba9ccf9b351b")
	if err != nil {
		t.Errorf("CancelTx failed unexpected error: %v\n", err)
		return
	}

	log.Infof("flag: %v", flag)

}

func TestValidateAddress(t *testing.T) {
	isVaild, err := tw.walletClient.ValidateAddress("46bf4426eb8142f58898ba9ccf9b351b")
	if err != nil {
		t.Errorf("vaild address failed unexpected error: %v\n", err)
		return
	}

	log.Infof("result: %v", isVaild)

}
