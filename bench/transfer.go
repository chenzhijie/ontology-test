package bench

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/ontio/ontology-crypto/keypair"
	ontSdk "github.com/ontio/ontology-go-sdk"
	ontSdkCom "github.com/ontio/ontology-go-sdk/common"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/types"
)

const (
	DEF_WALLET_PWD   = "pwd" //default wallet password
	NODES_ADDRS_FILE = "./addrs"
)

const (
	TRANSFER_ONT_DURATION     = 1   // transfer ont duration in second
	NO_ENOUGH_FOUND_MAX_CHECK = 600 // max check 0 balance times, if reach, stop the timer
)

type TestTransfer struct {
	defAccount     *account.Account
	sdk            *ontSdk.OntologySdk
	toAddrs        []common.Address
	repeat         int // repeat times for transfer to a address
	noEnoughFound  int
	stopTimerCh    chan bool
	lock           *sync.Mutex
	accountBalance *ontSdkCom.Balance
	tps            int // transaction per second
	amount         uint64
	rpcAddr        string
}

func NewTestTransfer() *TestTransfer {
	return &TestTransfer{}
}

func (this *TestTransfer) Start() {
	log.Infof("ont_tps:%d, amount:%d, rpc address:%s\n", this.tps, this.amount, this.rpcAddr)
	ret := this.initVars()
	if !ret {
		log.Error("init instance variable failed")
		return
	}
	timer := time.NewTicker(time.Duration(TRANSFER_ONT_DURATION * time.Second))
	for {
		select {
		case <-timer.C:
			this.transferOnt()
		case <-this.stopTimerCh:
			log.Info("stop timer because no enough found")
			timer.Stop()
			goto FINISHED
		}
	}
FINISHED:
	log.Info("finished")
}

func (this *TestTransfer) MultiTransfer() {

	this.sdk = ontSdk.NewOntologySdk()
	this.sdk.Rpc.SetAddress(this.rpcAddr)

	wallets := []string{"./bench/wallet1.dat", "./bench/wallet2.dat"}
	accs := make([]*account.Account, 0)
	pks := make([]keypair.PublicKey, 0, len(wallets))
	m := 2
	for _, w := range wallets {
		clientImpl, err := account.NewClientImpl(w)
		if clientImpl == nil {
			log.Errorf("clientImpl is nil")
			return
		}
		if err != nil {
			log.Errorf("import wallet failed")
			return
		}
		a, err := clientImpl.GetDefaultAccount([]byte("pwd"))
		if a == nil {
			log.Errorf("acc is nil")
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
	log.Infof("payer:%s ont:%d", payer.ToBase58(), balance.Ont)
	toAddr, err := common.AddressFromBase58("TA6rJ4vjeFmL8M7WGx6s4idEmbvDYLNjzc")
	toBalance, err := this.sdk.Rpc.GetBalance(toAddr)
	if err != nil {
		log.Errorf("addr from base 58 err:%s", err)
	}
	log.Infof("to:%s ont:%d", toAddr.ToBase58(), toBalance.Ont)
	txhash, err := this.sdk.Rpc.MultiSigTransfer(0, 30000, "ONT", accs, 2, toAddr, 10)
	if err != nil {
		log.Errorf("multi sig err:%s", err)
	}
	log.Infof("hash: %x", txhash)
}

func (this *TestTransfer) SetTps(tps int) {
	this.tps = tps
}

func (this *TestTransfer) SetAmount(amount uint64) {
	this.amount = amount
}

func (this *TestTransfer) SetRpc(rpc string) {
	this.rpcAddr = rpc
}

func (this *TestTransfer) initVars() bool {

	this.lock = &sync.Mutex{}

	this.toAddrs = getToAddrs()
	if this.toAddrs == nil || len(this.toAddrs) == 0 {
		log.Warnf("no transfer to address")
		return false
	}
	this.repeat = (int)(this.tps / len(this.toAddrs))
	log.Infof("Transfer address count:%d, each address repeat %d", len(this.toAddrs), this.repeat)
	this.noEnoughFound = 0
	this.stopTimerCh = make(chan bool, 1)

	this.sdk = ontSdk.NewOntologySdk()
	this.sdk.Rpc.SetAddress(this.rpcAddr)

	clientImpl, err := account.NewClientImpl("wallet.dat")
	if err != nil {
		log.Errorf("import wallet failed")
		return false
	}

	this.defAccount, err = clientImpl.GetDefaultAccount([]byte(DEF_WALLET_PWD))
	if err != nil {
		log.Errorf("client get default account failed")
		return false
	}
	// defAccount = account.NewAccount("")
	log.Infof("default account address:%v", this.defAccount.Address.ToBase58())
	this.accountBalance, err = this.sdk.Rpc.GetBalance(this.defAccount.Address)
	if err != nil {
		log.Errorf("get balance failed, error:%s", err)
		return false
	}

	return true
}

func (this *TestTransfer) transferOnt() {
	if !this.isBalanceEnough() {
		return
	}
	counter := 0
	for i := 0; i < this.repeat; i++ {
		for _, toAddr := range this.toAddrs {
			if !this.isBalanceEnough() {
				return
			}
			gasLimit := 30000 + i
			txHash, err := this.sdk.Rpc.Transfer(0, uint64(gasLimit), "ONT", this.defAccount, toAddr, this.amount)
			if err != nil {
				log.Errorf("transfer error:%s, txHash:%x", err, txHash)
				continue
			}

			counter++
			this.lock.Lock()
			this.accountBalance.Ont = this.accountBalance.Ont - this.amount
			fmt.Printf("time:%s, balance: %d\n", time.Now().Format("2006-01-02_15.04.05"), this.accountBalance.Ont)
			this.lock.Unlock()
		}
	}
}

func (this *TestTransfer) isBalanceEnough() bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.accountBalance.Ont == 0 || this.accountBalance.Ont < this.amount {
		log.Warnf("no enough ont, balance:%d", this.accountBalance.Ont)
		this.noEnoughFound++
		if this.noEnoughFound > NO_ENOUGH_FOUND_MAX_CHECK {
			this.stopTimerCh <- true
		}
		return false
	} else {
		return true
	}
}

// read address from file
func getToAddrs() []common.Address {
	file, err := ioutil.ReadFile(NODES_ADDRS_FILE)
	if err != nil {
		return nil
	}
	addrs := strings.Split(string(file), "\n")
	var addresses []common.Address
	for _, addr := range addrs {
		if len(addr) > 0 {
			toAddr, err := common.AddressFromBase58(addr)
			if err != nil {
				continue
			}
			addresses = append(addresses, toAddr)
		}
	}
	return addresses
}
