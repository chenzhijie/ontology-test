package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ontio/ontology-test/analysis"
	"github.com/ontio/ontology-test/bench"
	"github.com/ontio/ontology-test/cmd"
	"github.com/ontio/ontology/common/log"
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
		t.MultiSigTransfer()
	case cmd.CmdActionBatchAnalysis:
		txn := analysis.SumUpTxs(c.GetAnalysisPath())
		log.Infof("tx cnt:		%d", txn)
	case cmd.CmdActionInvalidTransfer:
		ty := c.GetInvalidTxType()
		t := bench.NewTestTransfer()
		t.SetRpc(c.GetRpc())
		t.InvokeInvalidTransaction(bench.InvalidTxType(ty))
	case cmd.CmdActionSignatureService:
		t := bench.NewTestTransfer()
		t.SetRpc(c.GetRpc())
		t.SignatureService()
	}
}
