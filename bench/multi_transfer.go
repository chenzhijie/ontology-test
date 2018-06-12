package bench

import (
	"time"

	"github.com/ontio/ontology-crypto/keypair"
	ontSdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/types"
)

const (
	WALLET_RECEIVER_ACC = "./bench/wallet.dat"
	WALLET_RECEIVER_PWD = "pwd"
)

func (this *TestTransfer) MultiTransfer() {

	this.sdk = ontSdk.NewOntologySdk()
	this.sdk.Rpc.SetAddress(this.rpcAddr)

	m := 2
	var wallets []string
	// 2-2 transfer
	// wallets = []string{"./bench/wallet1.dat", "./bench/wallet2.dat"}
	// this.multiTransferW(wallets, uint8(m), 4000)
	// 2-3 transfer
	wallets = []string{"./bench/wallet1.dat", "./bench/wallet2.dat", "./bench/wallet3.dat"}
	this.multiTransferW(wallets, uint8(m), 4000)
}

func (this *TestTransfer) multiTransferW(wallets []string, m uint8, tps int) {
	accs := make([]*account.Account, 0)
	pks := make([]keypair.PublicKey, 0, len(wallets))
	for _, w := range wallets {
		a, err := loadWallet(w, "pwd")
		if err != nil {
			log.Errorf("load wallet error: %s", err)
			return
		}
		accs = append(accs, a)
		pks = append(pks, a.PublicKey)
		log.Infof("addr:%s", a.Address.ToBase58())
	}
	payer, err := types.AddressFromMultiPubKeys(pks, int(m))
	if err != nil {
		log.Errorf("AddressFromMultiPubKeyserr:%s", err)
		return
	}
	balance, err := this.sdk.Rpc.GetBalance(payer)
	payerOnt := balance.Ont
	log.Infof("payer:%s ont:%d", payer.ToBase58(), balance.Ont)
	toAcc, err := loadWallet(WALLET_RECEIVER_ACC, WALLET_RECEIVER_PWD)
	toAddr := toAcc.Address
	toBalance, err := this.sdk.Rpc.GetBalance(toAddr)
	if err != nil {
		log.Errorf("addr from base 58 err:%s", err)
	}
	log.Infof("to:%s ont:%d", toAddr.ToBase58(), toBalance.Ont)
	timer := time.NewTicker(time.Duration(time.Second))
	gasLimit := 30000
	for {
		select {
		case <-timer.C:
			txPerRoutine := 200
			goRoutineCnt := tps / txPerRoutine
			amount := uint64(tps) * 1
			if payerOnt > 0 {
				for i := 0; i < goRoutineCnt; i++ {
					go func(id int) {
						for j := 0; j < txPerRoutine; j++ {
							gasLimit++
							txhash, err := this.sdk.Rpc.MultiSigTransfer(0, uint64(gasLimit), "ONT", accs, m, toAddr, 1)
							log.Infof("gasLimit:%d, hash: %x", gasLimit, txhash)
							if err != nil {
								log.Errorf("multi sig err:%s", err)
							}
						}
					}(i)
				}
				payerOnt -= amount
			}
		}
	}
}

func loadWallet(wallet, pwd string) (*account.Account, error) {
	clientImpl, err := account.NewClientImpl(wallet)
	if clientImpl == nil {
		log.Errorf("clientImpl is nil")
		return nil, err
	}
	a, err := clientImpl.GetDefaultAccount([]byte(pwd))
	if a == nil {
		log.Errorf("acc is nil")
		return nil, err
	}
	return a, nil
}
