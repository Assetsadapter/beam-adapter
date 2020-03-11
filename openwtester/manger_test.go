package openwtester

import (
	"fmt"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/blocktree/openwallet/owtp"
	"github.com/mr-tron/base58"
	"testing"
	"time"
)

func TestCreateLocalAddress(t *testing.T) {
	addrs, err := clientNode.CreateLocalWalletAddress(1, 2)
	if err != nil {
		t.Errorf("CreateRemoteWalletAddress failed unexpected error: %v\n", err)
		return
	}

	for i, a := range addrs {
		log.Infof("%d: %s", i, a)
	}
}

func TestCreateRemoteAddress(t *testing.T) {
	addrs, err := clientNode.CreateRemoteWalletAddress(1, 10)
	if err != nil {
		t.Errorf("CreateRemoteWalletAddress failed unexpected error: %v\n", err)
		return
	}

	for i, a := range addrs {
		log.Infof("%d: %s", i, a)
	}
	time.Sleep(30 * time.Second)
}

func TestGetWalletBalance(t *testing.T) {
	balanceLocal, err := clientNode.GetLocalWalletBalance()
	if err != nil {
		t.Errorf("GetLocalWalletBalance failed unexpected error: %v\n", err)
		return
	}

	balanceRemote, err := clientNode.GetRemoteWalletBalance()
	if err != nil {
		t.Errorf("GetRemoteWalletBalance failed unexpected error: %v\n", err)
		return
	}

	log.Infof("local balance: %+v", balanceLocal)
	log.Infof("remote balance: %+v", balanceRemote)

}

func TestGetLocalWalletAddress(t *testing.T) {
	addrs, err := clientNode.GetLocalWalletAddress()
	if err != nil {
		t.Errorf("CreateRemoteWalletAddress failed unexpected error: %v\n", err)
		return
	}

	for i, a := range addrs {
		log.Infof("%d: %s", i, a)
	}
}

func TestGetRemoteWalletAddress(t *testing.T) {
	addrs, err := clientNode.GetRemoteWalletAddress()
	if err != nil {
		t.Errorf("CreateRemoteWalletAddress failed unexpected error: %v\n", err)
		return
	}

	for _, a := range addrs {
		fmt.Printf("%s\n", a)
	}
}

func TestGetRemoteBlockByHeight(t *testing.T) {
	block, err := clientNode.GetRemoteBlockByHeight(1000)
	if err != nil {
		t.Errorf("CreateRemoteWalletAddress failed unexpected error: %v\n", err)
		return
	}

	log.Infof("%+v", block)
	time.Sleep(3 * time.Second)

}

func TestSendTransaction(t *testing.T) {

	rawTx := &openwallet.RawTransaction{
		Coin: openwallet.Coin{
			Symbol: "BEAM",
		},
		To: map[string]string{
			"191dfb3e032c47571914c8f5d5e270c027fab63aa4c5552390eeb096ec6153cc17d": "0.0000001",
		},
		FeeRate: "",
	}

	txdecoder := clientNode.TxDecoder
	//err := txdecoder.CreateRawTransaction(nil, rawTx)
	//if err != nil {
	//	t.Errorf("CreateRawTransaction failed unexpected error: %v\n", err)
	//	return
	//}
	//
	//err = txdecoder.SignRawTransaction(nil, rawTx)
	//if err != nil {
	//	t.Errorf("SignRawTransaction failed unexpected error: %v\n", err)
	//	return
	//}
	//
	//err = txdecoder.VerifyRawTransaction(nil, rawTx)
	//if err != nil {
	//	t.Errorf("VerifyRawTransaction failed unexpected error: %v\n", err)
	//	return
	//}

	tx, err := txdecoder.SubmitRawTransaction(nil, rawTx)
	if err != nil {
		t.Errorf("SubmitRawTransaction failed unexpected error: %v\n", err)
		return
	}

	log.Std.Info("tx: %+v", tx)
	log.Info("wxID:", tx.WxID)
	log.Info("txID:", rawTx.TxID)
}

func TestGetBalance(t *testing.T) {
	scanner := clientNode.Blockscanner
	balance, err := scanner.GetBalanceByAddress()
	if err != nil {
		t.Errorf("GetBalanceByAddress failed unexpected error: %v\n", err)
		return
	}
	log.Infof("balance: %+v", balance[0])
}

func TestStartSummaryWallet(t *testing.T) {

	txid, summaryAmount, feeAmount, err := clientNode.SummaryWalletProcess("22050821304db464e209b0dba622ffe3c711123471f0b300c7cc979df48e7a9a8a4")
	if err != nil {
		panic(err)
	}
	log.Info("txid=", txid, " summaryAmount=", summaryAmount, " feeAmount = ", feeAmount)

}

func TestGetKey(t *testing.T) {
	cert := owtp.NewRandomCertificate()
	log.Info(base58.Encode(cert.PrivateKeyBytes()))
	nodeID := owcrypt.Hash(cert.PublicKeyBytes(), 0, owcrypt.HASH_ALG_SHA256)
	log.Info(base58.Encode(nodeID))
}

func TestWalletManager_BackupWalletData(t *testing.T) {
	err := clientNode.BackupWalletData()
	if err != nil {
		t.Errorf("BackupWalletData failed unexpected error: %v\n", err)
		return
	}
}

func TestValidateAddress(t *testing.T) {
	isVaild, err := clientNode.ValidateAddress("b4ac9b088c4ea5f8b40f91113219216a596f082e27a809449c45add0c326e7dfb8")
	if err != nil {
		t.Errorf("vaild address failed unexpected error: %v\n", err)
		return
	}

	log.Infof("result: %v", isVaild)
}
