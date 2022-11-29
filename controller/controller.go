package main

import (
	"log"
	"fmt"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

var vmstats map[string]map[string]map[string]string	

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func updateMap(data map[string]map[string]map[string]map[string]string) {
	for vm, vmmap := range data {
		log.Printf("Received message: %s :: %s", vm, vmmap)
		if(vmstats[vm] == nil) {
			vmstats[vm] = make(map[string]map[string]string)
		}
		for key, element := range vmmap["value_diffs"] {
			fmt.Println("Key:", key, "=>", "Element:", element)
			vmstats[vm][key] = element
		}
		for key, element := range vmmap["added"] {
			fmt.Println("Key:", key, "=>", "Element:", element)
			vmstats[vm][key] = element    
		}
		for key, element := range vmmap["removed"] {
			fmt.Println("Key:", key, "=>", "Element:", element)
			delete(vmstats[vm], key)
		}
	}
}


func main() {
	
	vmstats = make(map[string]map[string]map[string]string)
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			var data map[string]map[string]map[string]map[string]string
			err := json.Unmarshal([]byte(d.Body), &data)
			if err != nil {
				panic(err)
			}
			updateMap(data);
			
			fmt.Println("-------------Map-------------")
			for k, v := range vmstats {
				fmt.Println(k, v)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}