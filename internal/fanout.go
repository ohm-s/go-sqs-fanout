package internal

import (
	"encoding/json"
	"github.com/ohm-s/go-sqs-fanout/pkg"
	log "github.com/sirupsen/logrus"
	"os"
	"net"
	"strconv"
	"time"
	"bytes"
)




func RunTcpListener(fanout pkg.Fanout)  (closeChannel chan int) {
	ln, err := net.Listen("tcp", pkg.GetFanoutAddr())
	if err != nil {
		log.WithField("addr", pkg.GetFanoutAddr()).Fatal("Could not bind to address")
		os.Exit(1)
	}
	closeChannel = make(chan int)
	go handleConnections(fanout, closeChannel, ln)
	return
}

func handleConnections(fanout pkg.Fanout, closeChannel <-chan int, listener net.Listener) {
	defer listener.Close()
	loop := true
	for loop {
		select {
		case <-closeChannel:
			listener.Close()
			loop = false
			break
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Error(err)
			} else {
				go handleRequest(fanout, conn)
			}
		}
	}
}

func handleRequest(fanout pkg.Fanout, conn net.Conn) {
	log.WithField("method", "handleRequest").Print("start handling")
	defer conn.Close()
	buf := make([]byte, pkg.MaxSqsMessageSize)
	conn.SetDeadline(time.Now().Add(time.Millisecond * 100))
	_, err := conn.Read(buf)
	if err != nil {
		log.WithField("method", "handleRequest").Error(err.Error())
		sendError(conn, err)
	} else {
		log.WithField("method", "handleRequest").Print("creating gopher")
		var message map[string]string
		buf = bytes.Trim(buf, "\x00")
		errJson := json.Unmarshal(buf, &message)
		if errJson != nil {
			sendError(conn, errJson)
		} else {
			queueName, foundName := message["queueName"]
			payload, foundPayload := message["payload"]
			if !foundName || !foundPayload {
				pkg.InvalidRequests.With(map[string]string{}).Inc()
				sendFailure(conn,  "queueName and payload are required")
			} else {
				var delaySeconds int64
				if delay, foundDelay := message["delay"]; foundDelay {
					delayValue, err := strconv.Atoi(delay)
					if err == nil && delayValue <= 900 && delayValue >= 0 {
						delaySeconds = int64(delayValue)
					}
				}
				log.WithField("method", "handleRequest").WithField("queue", queueName).WithField("payload", payload).Print("publish ")

				err := fanout.Publish(queueName, payload, delaySeconds)
				if err != nil {
					log.WithField("method", "handleRequest").WithField("queue", queueName).WithField("payload", payload).Error(err)
					sendError(conn, err)
				} else {
					sendSuccess(conn)
				}

			}
		}
	}
}



func sendError(conn net.Conn, err error) {
	resp, _ := json.Marshal(map[string]string{"status": "failed", "error": err.Error()})
	resp = append(resp, pkg.NewlineCharacter)
	conn.Write(resp)
}

func sendFailure(conn net.Conn, failure string) {
	resp, _ := json.Marshal(map[string]string{"status": "failed", "failure": failure})
	resp = append(resp, pkg.NewlineCharacter)
	conn.Write(resp)
}

func sendSuccess(conn net.Conn) {
	resp, _ := json.Marshal(map[string]string{"status": "ok"})
	resp = append(resp, pkg.NewlineCharacter)
	conn.Write(resp)
}