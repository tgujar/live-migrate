package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type cstats struct {
	mu         sync.Mutex
	Containers map[string]map[string]string
	AliveTime  time.Time
}

var vmstats map[string]cstats

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	var userIP string
	if len(r.Header.Get("CF-Connecting-IP")) > 1 {
		userIP = r.Header.Get("CF-Connecting-IP")
	} else if len(r.Header.Get("X-Forwarded-For")) > 1 {
		userIP = r.Header.Get("X-Forwarded-For")
	} else if len(r.Header.Get("X-Real-IP")) > 1 {
		userIP = r.Header.Get("X-Real-IP")
	} else {
		userIP = r.RemoteAddr
	}
	userIP = net.ParseIP(strings.Split(userIP, ":")[0]).String()
	cstat := vmstats[userIP]
	cstat.mu.Lock()
	cstat.AliveTime = time.Now()
	vmstats[userIP] = cstat
	log.Println(userIP)
	cstat.mu.Lock()
	w.WriteHeader(http.StatusOK)
}

func updateMap(data map[string]map[string]map[string]map[string]string) {
	for vm, vmmap := range data {
		log.Printf("Received message: %s :: %s", vm, vmmap)
		cstat := vmstats[vm]
		cstat.mu.Lock()
		if cstat.Containers == nil {
			cstat.Containers = make(map[string]map[string]string)
		}
		for key, element := range vmmap["value_diffs"] {
			fmt.Println("Key:", key, "=>", "Element:", element)
			cstat.Containers[key] = element
		}
		for key, element := range vmmap["added"] {
			fmt.Println("Key:", key, "=>", "Element:", element)
			cstat.Containers[key] = element
		}
		for key, element := range vmmap["removed"] {
			fmt.Println("Key:", key, "=>", "Element:", element)
			delete(cstat.Containers, key)
		}
		vmstats[vm] = cstat
		cstat.mu.Unlock()
	}
}

func main() {

	vmstats = make(map[string]cstats)
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
			updateMap(data)

			fmt.Println("-------------Map-------------")
			for k, v := range vmstats {
				fmt.Println(k, v)
			}
		}
	}()

	go func() {
		log.Println("Serving HeartBeat Server")
		http.HandleFunc("/heartbeat", heartbeatHandler)
		http.ListenAndServe("0.0.0.0:8090", nil)
	}()

	go func() {
		currtime := time.Now().Add(time.Minute * -1)
		for k, v := range vmstats {
			v.mu.Lock()
			if v.AliveTime.Before(currtime) {
				delete(vmstats, k)
			}
			v.mu.Unlock()
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
