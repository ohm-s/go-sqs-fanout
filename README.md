**Go SQS Fanout**

This is a fire and forget SQS proxy to minimize latency when sending messages to SQS queues from APIs   

Other solutions are available with its own unique set of  
* Composer library: https://async-aws.com/  this PHP library is the simplest way of sending asynchrononously to SQS however this will still need to block the PHP-FPM process.
* FluentBit/log forwarders: https://github.com/ixixi/fluent-plugin-sqs this solution leverages syslog to capture log data which represents SQS messages and forward it. This method hands over control of critical data to the logging daemon. 

Go SQS Fanout uses TCP messaging to receive SQS messages, handle the set up of SQS Queues if needed, then batching data sending to SQS

Env variable configuration
````
#required
fanout.queue.prefix=gopher_
fanout.addr="127.0.0.1:32122"
fanout.prom.addr="127.0.0.1:32123"
#optional, if metadata endpoint avaiable
aws.region="us-east-3"
#optional
aws.retries=3
aws.connection.pool=100
aws.connection.timeout=10
GIN_MODE=release

````

