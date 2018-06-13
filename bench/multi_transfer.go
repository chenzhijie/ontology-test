package bench

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/cmd/utils"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/types"
)

const (
	WALLET_RECEIVER_ACC = "./bench/wallet.dat"
	WALLET_RECEIVER_PWD = "pwd"
)

func (this *TestTransfer) MultiSigTransfer() {

	m := 2
	var wallets []string
	// 2-2 transfer
	// wallets = []string{"./bench/wallet1.dat", "./bench/wallet2.dat"}
	// this.multiSigTransferW(wallets, uint16(m), 400)
	// // 2-3 transfer
	wallets = []string{"./bench/wallet1.dat", "./bench/wallet2.dat", "./bench/wallet3.dat"}
	this.multiSigTransferW(wallets, uint16(m), 200)
}

func (this *TestTransfer) multiSigTransferW(wallets []string, m uint16, tps int) {
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
	balance, err := utils.GetBalance(payer.ToBase58())
	log.Infof("payer:%s ont:%s", payer.ToBase58(), balance.Ont)
	payerOnt, err := strconv.Atoi(balance.Ont)
	toAcc, err := loadWallet(WALLET_RECEIVER_ACC, WALLET_RECEIVER_PWD)
	toAddr := toAcc.Address
	toBalance, err := utils.GetBalance(toAddr.ToBase58())
	if err != nil {
		log.Errorf("addr from base 58 err:%s", err)
	}
	log.Infof("to:%s ont:%s", toAddr.ToBase58(), toBalance.Ont)
	if tps == 1 {
		txhash, err := transfer(0, 30000, accs, m, "ont", payer.ToBase58(), toAddr.ToBase58(), 10)
		if err != nil {
			log.Errorf("multi sig err:%s", err)
		}
		log.Infof("transfer one tx:%s", txhash)
		return
	}
	timer := time.NewTicker(time.Duration(time.Second))
	gasLimit := 30000
	for {
		select {
		case <-timer.C:
			txPerRoutine := 200
			goRoutineCnt := tps / txPerRoutine
			if payerOnt > 0 {
				for i := 0; i < goRoutineCnt; i++ {
					go func(id int) {
						for j := 0; j < txPerRoutine; j++ {
							gasLimit++
							txhash, err := transfer(0, uint64(gasLimit), accs, m, "ont", payer.ToBase58(), toAddr.ToBase58(), 1)
							log.Infof("gasLimit:%d, hash: %x", gasLimit, txhash)
							if err != nil {
								log.Errorf("multi sig err:%s", err)
							}
						}
					}(i)
				}
				payerOnt -= tps
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

//Transfer ont|ong from account to another account
func transfer(gasPrice, gasLimit uint64, signers []*account.Account, m uint16, asset, from, to string, amount uint64) (string, error) {
	transferTx, err := utils.TransferTx(gasPrice, gasLimit, asset, from, to, amount)
	if err != nil {
		return "", err
	}
	if len(signers) == 1 {
		err = utils.SignTransaction(signers[0], transferTx)
		if err != nil {
			return "", fmt.Errorf("SignTransaction error:%s", err)
		}
	} else {
		err = multiSignTransaction(transferTx, signers, m)
		if err != nil {
			return "", fmt.Errorf("SignTransaction error:%s", err)
		}
	}
	txHash, err := utils.SendRawTransaction(transferTx)
	if err != nil {
		return "", fmt.Errorf("SendTransaction error:%s", err)
	}
	return txHash, nil
}

//MultiSignTransaction multi sign to a transaction
func multiSignTransaction(tx *types.Transaction, signers []*account.Account, m uint16) error {
	if len(signers) == 0 {
		return fmt.Errorf("not enough signer")
	}
	n := len(signers)
	if int(m) > n {
		return fmt.Errorf("M:%d should smaller than N:%d", m, n)
	}

	pks := make([]keypair.PublicKey, 0, n)
	for _, signer := range signers {
		pks = append(pks, signer.PublicKey)
	}
	payer, err := types.AddressFromMultiPubKeys(pks, int(m))
	if err != nil {
		return fmt.Errorf("AddressFromMultiPubKeys error:%s", payer)
	}
	tx.Payer = payer

	txHash := tx.Hash()
	sigData := make([][]byte, 0, m)
	for i := 0; i < n; i++ {
		signer := signers[i]
		if i >= int(m) {
			break
		}
		sig, err := utils.Sign(txHash.ToArray(), signer)
		if err != nil {
			return fmt.Errorf("sign error:%s", err)
		}
		sigData = append(sigData, sig)
	}
	sig := &types.Sig{
		PubKeys: pks,
		M:       m,
		SigData: sigData,
	}
	tx.Sigs = []*types.Sig{sig}

	return nil
}
