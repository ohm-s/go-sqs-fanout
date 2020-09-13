package test

import (
	"bufio"
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/ohm-s/go-sqs-fanout/internal"
	"github.com/ohm-s/go-sqs-fanout/pkg"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"
)


func TestSocket(t *testing.T) {
	t.Log("starting up server for tests")
	closeHttp := make(chan int)
	ready := internal.SetupProm(closeHttp)
	<- ready
	fanout := pkg.NewFanout(func() *sqs.SQS {
		return pkg.NewSqsClient()
	})
	fanout.Run()

	internal.RunTcpListener(fanout)
	t.Log("starting client tests")
	conn, err := net.Dial("tcp", pkg.GetFanoutAddr())
	if err != nil {
		t.Error(err)
	}
	conn.SetDeadline(time.Now().Add(time.Second * 2))
	data := map[string]interface{} {"custom": "payload"}
	buffer, _ := json.Marshal(data)
	_, errWrite := conn.Write(buffer)
	if errWrite != nil {
		t.Error(errWrite)
	}

	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Error(err)
	}

	t.Log(netData)
 	var testObj map[string]interface{}
	errJson := json.Unmarshal([]byte(netData), &testObj)
	if errJson != nil {
		t.Error(errJson)
	}
	resp, err := http.Get("http://" + pkg.GetPromAddr() + "/metrics")
	if err != nil {
		t.Error(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	t.Log(string(body))

	fanout.Shutdown()
}