package pkg

import (
	"encoding/json"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

var fanoutAddr string
var awsRegion string
var awsRetries int
var awsDisableSsl bool
var awsAccountId string
var awsConnectionPool int
var awsConnetionTimeout int64
var promAddr string
var fanoutDebugMode bool
var fanoutQueuePrefix string


func init() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Warn("Could not load .env ")
	}
	fanoutAddr = os.Getenv("fanout.addr")
	if fanoutAddr == "" {
		log.Fatal("Fanout address not provided")
		os.Exit(1)
	}

	promAddr = os.Getenv("fanout.prom.addr")
	fanoutQueuePrefix = os.Getenv("fanout.queue.prefix")

	fanoutDebug := os.Getenv("fanout.debug")
	if fanoutDebug != "" && fanoutDebug != "0" {
		fanoutDebugMode = true
	}

	disableSsl := os.Getenv("aws.disablessl")
	if disableSsl != "" && disableSsl != "0" {
		awsDisableSsl = true
	}

	timeout := os.Getenv("aws.connection.timeout")
	numTimeout, err := strconv.Atoi(timeout)
	if err != nil {
		awsConnetionTimeout = 10
	} else {
		awsConnetionTimeout = int64(numTimeout)
	}

	pool := os.Getenv("aws.connection.pool")
	numPool, err := strconv.Atoi(pool)
	if err != nil {
		awsConnectionPool = 100
	} else {
		awsConnectionPool = numPool
	}


	retries := os.Getenv("aws.retries")
	numRetries, err := strconv.Atoi(retries)
	if err != nil {
		awsRetries = 0
	} else {
		awsRetries = numRetries
	}

	awsRegion = os.Getenv("aws.region")
	awsAccountId = os.Getenv("aws.accountid")

	if awsRegion == "" {
		client := http.Client{
			Timeout: 100 * time.Millisecond,
		}

		resp, err := client.Get(awsMetadataEndpoint)

		if err != nil {
			log.WithField("error", err).Error("AWS region is not provided. Could not retrieve aws region from metadata endpoint.")
			os.Exit(1)
		}
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.WithField("error", err).Error("AWS region is not provided. Malformed aws region from metadata endpoint.")
			os.Exit(1)
		}

		var identityDocument map[string]interface{}

		err = json.Unmarshal(buf, &identityDocument)

		if err != nil {
			log.WithField("error", err).Error("AWS region is not provided. Could not parse identity document.")
			os.Exit(1)
		}

		region, found := identityDocument["region"].(string)
		if !found {
			log.WithField("error", err).Error("AWS region is not provided. unknown state.")
			os.Exit(1)
		} else {
			awsRegion = region
		}

		if awsAccountId == "" {
			accountId, found := identityDocument["accountId"].(string)
			if !found {
				log.WithField("error", err).Error("AWS accountId is not provided. unknown state.")
				os.Exit(1)
			} else {
				awsAccountId = accountId
			}
		}

	}
}

func GetFanoutAddr() string {
	return fanoutAddr
}

func GetPromAddr() string {
	return promAddr
}

func GetFanoutQueuePrefix() string {
	return fanoutQueuePrefix
}

func GetAwsConnectionTimeout() time.Duration {
	return time.Duration(awsConnetionTimeout) * time.Second
}