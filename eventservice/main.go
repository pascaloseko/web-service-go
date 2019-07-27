package main

import (
	"flag"
	"fmt"
	"log"
	"github.com/pascaloseko/web-service-go/eventservice/rest"
	"github.com/pascaloseko/web-service-go/lib/configuration"
	"github.com/pascaloseko/web-service-go/lib/msgqueue"
	msgqueue_amqp "github.com/pascaloseko/web-service-go/lib/msgqueue/amqp"
	"github.com/pascaloseko/web-service-go/lib/msgqueue/kafka"
	"github.com/pascaloseko/web-service-go/lib/persistence/dblayer"

	"github.com/Shopify/sarama"
	"github.com/streadway/amqp"
)

func main() {
	var eventEmitter msgqueue.EventEmitter

	confPath := flag.String("conf", `.\configuration\config.json`, "flag to set the path to the configuration json file")
	flag.Parse()
	//extract configuration
	config, _ := configuration.ExtractConfiguration(*confPath)

	switch config.MessageBrokerType {
	case "amqp":
		conn, err := amqp.Dial(config.AMQPMessageBroker)
		if err != nil {
			panic(err)
		}

		eventEmitter, err = msgqueue_amqp.NewAMQPEventEmitter(conn, "events")
		if err != nil {
			panic(err)
		}
	case "kafka":
		conf := sarama.NewConfig()
		conf.Producer.Return.Successes = true
		conn, err := sarama.NewClient(config.KafkaMessageBrokers, conf)
		if err != nil {
			panic(err)
		}

		eventEmitter, err = kafka.NewKafkaEventEmitter(conn)
		if err != nil {
			panic(err)
		}
	default:
		panic("Bad message broker type: " + config.MessageBrokerType)
	}

	fmt.Println("Connecting to database")
	dbhandler, _ := dblayer.NewPersistenceLayer(config.Databasetype, config.DBConnection)
	//RESTful API start
	httpErrChan, httptlsErrChan := rest.ServeAPI(config.RestfulEndpoint, config.RestfulTLSEndpoint, dbhandler, eventEmitter)
	select {
	case err := <-httpErrChan:
		log.Fatal("HTTP ERROR: ", err)
	case err := <-httptlsErrChan:
		log.Fatal("HTTPS ERROR: ", err)
	}
}
