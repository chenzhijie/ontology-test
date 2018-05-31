package main

import (
	"fmt"
	"os"
	"time"

	"io/ioutil"
	"strings"
	"sync"

	ontSdk "github.com/ontio/ontology-go-sdk"
	ontSdkCom "github.com/ontio/ontology-go-sdk/common"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/urfave/cli"
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

func runApp() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "tps",
			Value: 100,
			Usage: "transaction per second",
		},
		cli.IntFlag{
			Name:  "amount",
			Value: 1,
			Usage: "transfer amount",
		},
		cli.IntFlag{
			Name:  "rpcport",
			Value: 30336,
			Usage: "local rpc server port",
		},
	}
	app.Action = func(c *cli.Context) error {
		ONT_TPS = c.Int("tps")
		TRANSFER_AMOUNT = c.Uint64("amount")
		LOCAL_RPC_ADDRESS = fmt.Sprintf("http://localhost:%d", c.Int("rpcport"))
		log.Infof("ont_tps:%d, amount:%d, rpc address:%s\n", ONT_TPS, TRANSFER_AMOUNT, LOCAL_RPC_ADDRESS)
		start()
		return nil
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func start() {
	ret := initVars()
	if !ret {
		log.Error("init instance variable failed")
		return
	}
	timer := time.NewTicker(time.Duration(TRANSFER_ONT_DURATION * time.Second))
	for {
		select {
		case <-timer.C:
			transferOnt()
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

	lock = &sync.Mutex{}

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

	clientImpl, err := account.NewClientImpl("wallet.dat")
	if err != nil {
		log.Errorf("import wallet failed")
		return false
	}

	defAccount, err = clientImpl.GetDefaultAccount([]byte(DEF_WALLET_PWD))
	if err != nil {
		log.Errorf("client get default account failed")
		return false
	}
	// defAccount = account.NewAccount("")
	log.Infof("default account address:%v", defAccount.Address.ToBase58())
	accountBalance, err = sdk.Rpc.GetBalance(defAccount.Address)
	if err != nil {
		log.Errorf("get balance failed, error:%s", err)
		return false
	}

	return true
}

func transferOnt() {
	if !isBalanceEnough() {
		return
	}
	counter := 0
	for _, toAddr := range toAddrs {
		for i := 0; i < repeat; i++ {
			if !isBalanceEnough() {
				return
			}
			gasLimit := 30000 + i
			txHash, err := sdk.Rpc.Transfer(0, uint64(gasLimit), "ONT", defAccount, toAddr, TRANSFER_AMOUNT)
			if err != nil {
				log.Errorf("transfer error:%s", err)
				continue
			}

			counter++

			lock.Lock()
			accountBalance.Ont = accountBalance.Ont - TRANSFER_AMOUNT
			log.Infof("%d: txHash:%x, to:%s, remain:%d", counter, txHash.ToArray(), toAddr.ToBase58(), accountBalance.Ont)
			lock.Unlock()
		}
	}
}

func isBalanceEnough() bool {
	lock.Lock()
	defer lock.Unlock()
	if accountBalance.Ont == 0 || accountBalance.Ont < TRANSFER_AMOUNT {
		log.Warnf("no enough ont, balance:%d", accountBalance.Ont)
		noEnoughFound++
		if noEnoughFound > NO_ENOUGH_FOUND_MAX_CHECK {
			stopTimerCh <- true
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
		toAddr, err := common.AddressFromBase58(addr)
		if err != nil {
			continue
		}
		addresses = append(addresses, toAddr)
	}
	return addresses
}
