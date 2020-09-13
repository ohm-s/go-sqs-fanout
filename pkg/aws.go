package pkg

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"net"
	"net/http"
	"time"
)


const MaxSqsMessageSize =  262144
const MaxSqsBatchSize = 10
const NewlineCharacter = 10
const awsMetadataEndpoint = "http://169.254.169.254/latest/dynamic/instance-identity/document"

func NewSqsClient() *sqs.SQS {
	timeoutBaseDuration := time.Duration(awsConnetionTimeout)
	timeout := timeoutBaseDuration * time.Second
	defaultTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   timeoutBaseDuration * time.Second,
			KeepAlive: 6 * timeoutBaseDuration * time.Second,
		}).DialContext,
		MaxIdleConns:        awsConnectionPool,
		MaxIdleConnsPerHost: awsConnectionPool,
	}
	client:= &http.Client{
		Transport: defaultTransport,
		Timeout:   timeout,
	}
	logLevel := aws.LogLevel(0)
	if fanoutDebugMode {
		logLevel = aws.LogLevel(4104)
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String(awsRegion),
			DisableSSL: aws.Bool(awsDisableSsl),
			MaxRetries: aws.Int(awsRetries),
			HTTPClient: client,
			LogLevel: logLevel,
		},
	}))
	obj := sqs.New(sess)
	return obj
}