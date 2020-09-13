package pkg

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type fanoutMessage struct {
	queueName string
	uuid string
	payload string
	delaySeconds int64
}
type readyStruct struct {
	ready bool
}
type fanout struct {
	started *readyStruct
	interruptChannel chan int
	messageChannel chan fanoutMessage
	channels map[string]chan fanoutMessage
	queueUrlMap map[string]*string
	sqsClientProvider func()*sqs.SQS
	pendingOps *int64
	mutex sync.RWMutex
	shutdown chan int
	queues map[string]string
	waitOperations sync.WaitGroup
}

type Fanout interface {
	Run()
	Shutdown()
	ShutdownComplete() chan int
	Publish(queueName string, payload string, delaySeconds int64) error
}

func NewFanout(sqsProvider func()*sqs.SQS) Fanout {
	f := fanout{
		messageChannel: make(chan fanoutMessage, 100),
		sqsClientProvider: sqsProvider,
		mutex: sync.RWMutex{},
		channels: map[string]chan fanoutMessage{},
		queueUrlMap: map[string]*string{},
		shutdown: make(chan int),
		interruptChannel: make(chan int),
		started: &readyStruct{
			ready: false,
		},
	}
	return f
}

func (f fanout) Publish(queueName string, payload string, delaySeconds int64) error {
	if !f.started.ready {
		return fmt.Errorf("Not started")
	}
	f.messageChannel <- fanoutMessage{queueName: queueName, payload: payload, delaySeconds: delaySeconds}
	return nil
}


func (f fanout) ShutdownComplete() chan int {
	return f.shutdown
}


func (f fanout) Run() {
	if !f.started.ready {
		f.interruptChannel = make(chan int)
		go func() {
			for {
				select {
					case <-f.interruptChannel:
					break
					case message := <- f.messageChannel:
						f.waitOperations.Add(1)
						f.process(message)
					default:
						<- time.After(time.Millisecond)
				}
			}
			log.Print("for loop exited. Marking server not ready")
			f.started.ready = false
		}()
		log.Print("Fanout ready")
		f.started.ready = true
	}
}

func (f fanout) Shutdown() {
	if f.started.ready {
		close(f.interruptChannel)
		f.started.ready = false
		f.mutex.Lock()
		defer f.mutex.Unlock()
		for _, channel := range f.channels {
			close(channel)
		}

	}
	f.waitOperations.Wait()
	close(f.shutdown)
}



func (f fanout)  process(message fanoutMessage) {
	FanoutTotal.With(map[string]string{"queue_name": message.queueName}).Inc()
	f.mutex.RLock()
	if item, found := f.channels[message.queueName]; found {
		f.mutex.RUnlock()
		item <- message
	} else {
		f.mutex.RUnlock()
		fanoutChannel := make(chan fanoutMessage, 10)
		f.mutex.Lock()
		defer f.mutex.Unlock()
		f.channels[message.queueName] = fanoutChannel

		queueName := message.queueName
		payloadInitial := message.payload
		timeDelayInitial := message.delaySeconds

		log.WithField("queue", queueName).Print("Launching a gopher")

		// start handling messages
		go func(initialPayload fanoutMessage) {
			sqsClient := f.sqsClientProvider()
			queueName = initialPayload.queueName
			payloadInitial = initialPayload.payload
			timeDelayInitial = initialPayload.delaySeconds
			list, err := sqsClient.ListQueues(&sqs.ListQueuesInput{
				QueueNamePrefix: aws.String(GetFanoutQueuePrefix() + queueName),
			})
			fmt.Printf("+%o \n", list)
			if err != nil {
				log.WithField("queue", queueName).WithField("operation", "listQueues").Error(err)
			}
			var url *string
			if len(list.QueueUrls) == 0 {
				output, err := sqsClient.CreateQueue(&sqs.CreateQueueInput{
					QueueName:  aws.String(GetFanoutQueuePrefix() + queueName),
				})
				if err == nil {
					fmt.Printf("+%o \n", output)
					url = output.QueueUrl
				} else {
					log.WithField("queue", queueName).WithField("operation", "createQueue").Error(err)
				}

			} else {
				url = list.QueueUrls[0]
			}

			log.WithField("queue", queueName).WithField("url", url).Print("fetched sqs url")

			if err != nil {
				log.WithField("queue", queueName).Error(err)
			}
			if url != nil && *url != "" {
				f.mutex.Lock()
				f.queueUrlMap[queueName] = url
				// run error logger in case we fail to create the queue
				for  message  := range fanoutChannel {
					log.WithField("queue", queueName).WithField("handler", "queueFailed").WithField("error", err).Warn(message)
				}
			} else {
				// run batch submit handler
				messages := make([]fanoutMessage, 0)
				messages = append(messages, fanoutMessage{
					queueName: queueName,
					payload: payloadInitial,
					delaySeconds: timeDelayInitial,
				})
				batch :=  make([]*sqs.SendMessageBatchRequestEntry, 0)
				batchedId := fmt.Sprintf("%d::%d::%s", time.Now().Nanosecond(), time.Now().Second(), queueName)
				batch = append(batch, &sqs.SendMessageBatchRequestEntry{
					Id: aws.String(batchedId),
					MessageBody: aws.String(initialPayload.payload),
					DelaySeconds: aws.Int64(initialPayload.delaySeconds),
				})

				payloadSize := 0
				maxPayloadSize := 0.9 * MaxSqsMessageSize
				log.WithField("queue", initialPayload.queueName).Print("starting to process stream")
				for {
					select {
						case message := <-fanoutChannel:
							messages = append(messages, message)
							if len(messages) > 0 {
								for _, message := range messages {
									if float64(len(message.payload) + payloadSize) > maxPayloadSize || len(batch) + 1 > MaxSqsBatchSize {
										sendMessages(f, sqsClient, url, batch, message.queueName)
										maxPayloadSize = 0
										batch =  make([]*sqs.SendMessageBatchRequestEntry, 0)
									}
									batchedId := fmt.Sprintf("%d::%d::%s", time.Now().Nanosecond(), time.Now().Second(), queueName)
									batch = append(batch, &sqs.SendMessageBatchRequestEntry{
										Id: aws.String(batchedId),
										MessageBody: aws.String(message.payload),
										DelaySeconds: aws.Int64(message.delaySeconds),
									})
								}
							}
					default:
						if len(batch) > 0 {
							go sendMessages(f, sqsClient, url, batch, initialPayload.queueName)
							maxPayloadSize = 0
							batch =  make([]*sqs.SendMessageBatchRequestEntry, 0)
						}
						<- time.After(time.Millisecond * 5)
					}
				}
			}
		}(message)
	}


}

func sendMessages(fn fanout, client *sqs.SQS, queueUrl *string, batch []*sqs.SendMessageBatchRequestEntry, queueName string) {
	log.WithField("queueUrl", queueUrl).Print("Sumbitnnng dataaaa ")
	defer fn.waitOperations.Add(-len(batch))
	_, err := client.SendMessageBatch(&sqs.SendMessageBatchInput{
		QueueUrl: queueUrl,
		Entries: batch,
	})
	if err != nil {
		log.WithField("error", err).WithField("shouldReProcess", 1).Error(batch)
		FanoutFailure.With(map[string]string{"queue_name": queueName}).Add(float64(len(batch)))
	} else {
		FanoutSuccess.With(map[string]string{"queue_name": queueName}).Add(float64(len(batch)))
	}
}
