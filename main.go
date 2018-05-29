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

const (
	LOCAL_RPC_ADDRESS = "http://localhost:30336"
)

const (
	TRANSFER_ONT_DURATION = 1 // transfer ont duration in second
	TRANSFER_AMOUNT       = 10
)

func main() {
	toAddrs = getToAddrs()
	if toAddrs == nil || len(toAddrs) == 0 {
		log.Warnf("no transfer to address")
		return
	}
	sdk = ontSdk.NewOntologySdk()
	sdk.Rpc.SetAddress(LOCAL_RPC_ADDRESS)
	log.InitLog(0, log.PATH, log.Stdout)
	defAccount = account.NewAccount("")
	log.Infof("default account address:%v", defAccount.Address.ToBase58())
	timer := time.NewTicker(time.Duration(TRANSFER_ONT_DURATION * time.Second))
	for {
		select {
		case <-timer.C:
			transferOnt()
		}
	}
}

func transferOnt() {
	bal, err := sdk.Rpc.GetBalance(defAccount.Address)
	if err != nil {
		log.Errorf("get balance failed, error:%s", err)
	}
	if bal.Ont == 0 || bal.Ont < TRANSFER_AMOUNT {
		log.Warnf("no enough ont, balance:%d", bal.Ont)
		return
	}
	for _, toAddr := range toAddrs {
		txHash, err := sdk.Rpc.Transfer(0, 30000, "ONT", defAccount, toAddr, TRANSFER_AMOUNT)
		if err != nil {
			log.Errorf("transfer error:%s", err)
			continue
		}
		log.Infof("tx success txHash:%x", txHash.ToArray())
	}
}

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
