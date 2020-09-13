package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/ohm-s/go-sqs-fanout/internal"
	"github.com/ohm-s/go-sqs-fanout/pkg"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{TimestampFormat: time.RFC3339,PrettyPrint: false, FieldMap: log.FieldMap{
		log.FieldKeyTime:  "@timestamp",
		log.FieldKeyLevel: "@level",
		log.FieldKeyMsg:   "@message",
		log.FieldKeyFunc:  "@caller",
	}})
}


func main() {
	httpCloseChannel := make(chan int)
	httpReadyChannel := internal.SetupProm(httpCloseChannel)
	<-httpReadyChannel

	log.Print("Attempting to start server on host: " + pkg.GetFanoutAddr())
	sqsClient := pkg.NewSqsClient()
	_, err := sqsClient.ListQueues(&sqs.ListQueuesInput{
		MaxResults: aws.Int64(1),
	})
	if err != nil { // verify connectivity
		log.WithField("method", "main").Panic(err)
		return
	} else {
		log.WithField("method", "main").Print("Connectivity ok")
	}

	_, err = sqsClient.ListQueues(&sqs.ListQueuesInput{
		MaxResults: aws.Int64(2),
	})
	if err != nil { // verify connectivity
		log.WithField("method", "main").Panic(err)
		return
	} else {
		log.WithField("method", "main").Print("Connectivity ok")
	}

	go func() {
		<- time.After(20 * time.Second)
		_, err = sqsClient.ListQueues(&sqs.ListQueuesInput{
			MaxResults: aws.Int64(2),
		})
		if err != nil { // verify connectivity
			log.WithField("method", "main").Panic(err)
			return
		} else {
			log.WithField("method", "main").Print("Connectivity ok")
		}

	}()

	fanout := pkg.NewFanout(func() *sqs.SQS {
		return pkg.NewSqsClient()
	})
	fanout.Run()

	shutdownComplete := fanout.ShutdownComplete()

	closeChannel := internal.RunTcpListener(fanout)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Print("Closing connections and flushing messages")
	close(closeChannel)
	<- time.After(pkg.GetAwsConnectionTimeout())
	fanout.Shutdown()
	// Enable queues to be flushed
	<- shutdownComplete
	if pkg.GetPromAddr() != "" {
		close(httpCloseChannel)
	}
	log.Print("Shutdown success")
}


