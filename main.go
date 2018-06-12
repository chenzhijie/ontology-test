package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"./analysis"
	"./bench"
	"./cmd"
	ontSdk "github.com/ontio/ontology-go-sdk"
	ontSdkCom "github.com/ontio/ontology-go-sdk/common"

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
var lock *sync.Mutex
var accountBalance *ontSdkCom.Balance

var ONT_TPS int // transaction per second
var TRANSFER_AMOUNT uint64
var LOCAL_RPC_ADDRESS string

const (
	DEF_WALLET_PWD   = "pwd" //default wallet password
	NODES_ADDRS_FILE = "./addrs"
)

const (
	TRANSFER_ONT_DURATION     = 1   // transfer ont duration in second
	NO_ENOUGH_FOUND_MAX_CHECK = 600 // max check 0 balance times, if reach, stop the timer
)

func main() {
	log.InitLog(0, log.PATH, log.Stdout)
	runApp()
}

func CheckHash(file string) {
	con, err := ioutil.ReadFile(fmt.Sprintf("./Log/%s", file))
	ret := strings.Split(string(con), "\n")
	if err != nil {

	}
	var hashes []string
	for _, line := range ret {
		if strings.Index(line, "hash") != -1 {
			hash := strings.Split(line, "hash: ")
			hashes = append(hashes, hash[1])
		}
	}
	for i, hash := range hashes {
		for j, h := range hashes {
			if hash == h && i != j {
				fmt.Println(hash)
			}
		}
	}
	fmt.Printf("done:%d\n", len(hashes))
}

func runApp() {
	c := cmd.NewCmd()
	c.Run()
	switch c.GetAction() {
	case cmd.CmdActionBatchTransfer:
		t := bench.NewTestTransfer()
		t.SetTps(c.GetOntTPS())
		t.SetAmount(c.GetAmount())
		t.SetRpc(c.GetRpc())
		t.Start()
	case cmd.CmdActionMutilTransfer:
		t := bench.NewTestTransfer()
		t.SetTps(c.GetOntTPS())
		t.SetAmount(c.GetAmount())
		t.SetRpc(c.GetRpc())
		t.MultiTransfer()
	case cmd.CmdActionBatchAnalysis:
		txn := analysis.SumUpTxs(c.GetAnalysisPath())
		log.Infof("tx cnt:		%d", txn)
	}
}
