package main

import (
	"io/ioutil"
	"strings"
	"time"

	ontSdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
)

var defAccount *account.Account
var sdk *ontSdk.OntologySdk
var toAddrs []common.Address
var repeat int // repeat times for transfer to a address
var noEnoughFound int
var stopTimerCh chan bool

const (
	LOCAL_RPC_ADDRESS = "http://localhost:30336"
)

const (
	TRANSFER_ONT_DURATION     = 1 // transfer ont duration in second
	TRANSFER_AMOUNT           = 1
	ONT_TPS                   = 200 // transaction per second
	NO_ENOUGH_FOUND_MAX_CHECK = 600 // max check 0 balance times, if reach, stop the timer
)

func main() {
	ret := initVars()
	if !ret {
		log.Error("init instance variable failed")
		return
	}
	timer := time.NewTicker(time.Duration(TRANSFER_ONT_DURATION * time.Second))
	for {
		select {
		case <-timer.C:
			go transferOnt()
		case <-stopTimerCh:
			log.Info("stop timer because no enough found")
			timer.Stop()
			goto FINISHED
		}
	}
FINISHED:
	log.Info("finished")
}

func initVars() bool {
	log.InitLog(0, log.PATH, log.Stdout)

	toAddrs = getToAddrs()
	if toAddrs == nil || len(toAddrs) == 0 {
		log.Warnf("no transfer to address")
		return false
	}
	repeat = (int)(ONT_TPS / len(toAddrs))
	log.Infof("Transfer address count:%d, each address repeat %d", len(toAddrs), repeat)
	noEnoughFound = 0
	stopTimerCh = make(chan bool, 1)

	sdk = ontSdk.NewOntologySdk()
	sdk.Rpc.SetAddress(LOCAL_RPC_ADDRESS)

	defAccount = account.NewAccount("")
	log.Infof("default account address:%v", defAccount.Address.ToBase58())
	return true
}

func transferOnt() {
	bal, err := sdk.Rpc.GetBalance(defAccount.Address)
	if err != nil {
		log.Errorf("get balance failed, error:%s", err)
	}
	if bal.Ont == 0 || bal.Ont < TRANSFER_AMOUNT {
		log.Warnf("no enough ont, balance:%d", bal.Ont)
		noEnoughFound++
		if noEnoughFound > NO_ENOUGH_FOUND_MAX_CHECK {
			stopTimerCh <- true
		}
		return
	}
	counter := 0
	for _, toAddr := range toAddrs {
		for i := 0; i < repeat; i++ {
			txHash, err := sdk.Rpc.Transfer(0, 30000, "ONT", defAccount, toAddr, TRANSFER_AMOUNT)
			if err != nil {
				log.Errorf("transfer error:%s", err)
				continue
			}
			log.Infof("%d: txHash:%x, to:%s", counter, txHash.ToArray(), toAddr.ToBase58())
			counter++
		}
	}
}

// read address from file
func getToAddrs() []common.Address {
	file, err := ioutil.ReadFile("./addrs")
	if err != nil {
		return nil
	}
	addrs := strings.Split(string(file), "\n")
	var addresses []common.Address
	for _, addr := range addrs {
		toAddr, err := common.AddressFromBase58(addr)
		if err != nil {
			continue
		}
		addresses = append(addresses, toAddr)
	}
	return addresses
}
