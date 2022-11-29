package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var mu sync.Mutex

type Cstats struct {
	Containers map[string]map[string]string
	AliveTime  time.Time
}

var VMstats map[string]Cstats

func scheduleMigration(vm_container_map map[string]map[string]float64, target float64) map[string]map[string]string {

	vm_total := make(map[string]float64)
	vm_mapping := make(map[string]string)
	total_vms := len(vm_container_map)
	vms := make([]string, 0, len(vm_container_map))

	container_init_vms := make(map[string]string)

	vm_index := make(map[string]int)

	vms_overused_queue := make([]string, 0)

	count := 0
	for key, element := range vm_container_map {

		vm_total[key] = 0.0
		for cont_id, cpu_perc := range element {
			vm_total[key] = vm_total[key] + cpu_perc
			container_init_vms[cont_id] = key
		}
		if vm_total[key] > target {
			vms_overused_queue = append(vms_overused_queue, key)
		}
		vms = append(vms, key)
		vm_index[key] = count
		count++
	}

	fmt.Println("Printing VM's totals initial:")
	for k, v := range vm_total {
		fmt.Println("VM: ", k, ", Total Usage: ", v)
	}
	fmt.Println("")

	overused_redundancy := 0

	for len(vms_overused_queue) > 0 {
		key := vms_overused_queue[0]
		element := vm_container_map[key]

		// fmt.Println("Key: ", key)
		// for k,v := range element{
		// 	fmt.Println("Container: ", k, " Usage: ", v)
		// }

		if vm_total[key] > target {
			keys := make([]string, 0, len(element))

			for key_elem := range element {
				if element[key_elem] >= 0.009 {
					keys = append(keys, key_elem)
				}
			}

			sort.SliceStable(keys, func(i, j int) bool {
				return element[keys[i]] >= element[keys[j]]
			})

			// fmt.Println("VM: ", key)
			// for _, k := range keys {
			// 	fmt.Println("Container: ", k)
			// }

			for _, key_elem := range keys {
				if vm_total[key] <= target {
					break
				}

				isStillThere := true
				a := vm_index[key]
				initial_a := a

				for isStillThere {

					a = (a + 1) % total_vms
					if a == initial_a {
						break
					}

					if (vm_total[vms[a]]+element[key_elem] < 1.00) && ((vm_total[vms[a]] + element[key_elem]) < vm_total[key]) {

						vm_mapping[key_elem] = vms[a]
						vm_container_map[vms[a]][key_elem] = element[key_elem]
						vm_total[vms[a]] = vm_total[vms[a]] + element[key_elem]
						if vm_total[vms[a]] > target {
							vms_overused_queue = append(vms_overused_queue, vms[a])
						}
						vm_total[key] = vm_total[key] - element[key_elem]

						isStillThere = false
						delete(vm_container_map[key], key_elem)
					}
				}
			}
		}

		vms_overused_queue = vms_overused_queue[1:]
		if vm_total[key] > target {
			vms_overused_queue = append(vms_overused_queue, key)
			overused_redundancy++
		} else {
			overused_redundancy = 0
		}

		if overused_redundancy > len(vms_overused_queue) {
			break
		}

	}

	fmt.Println("Printing VM's totals final:")
	for k, v := range vm_total {
		fmt.Println("VM: ", k, ", Total Usage: ", v)
	}
	fmt.Println("")
	fmt.Println("Target: ", target)
	fmt.Println("")

	vm_map_from_to := make(map[string]map[string]string)

	for k, v := range vm_mapping {
		init_vm := container_init_vms[k]
		vm_map_from_to[k] = make(map[string]string)
		vm_map_from_to[k][init_vm] = v
	}

	return vm_map_from_to

}

// Map from string to Cstats
// Cstats: {"84e30906595e": {"name": "peaceful_hoover", "cpu_util": "2.01%"}}
func UpdateMigration(vm_map map[string]Cstats) map[string]map[string]string {

	target := 0.60

	var vm_cont_map map[string]map[string]float64 = make(map[string]map[string]float64)
	for vm_id, cstat_val := range vm_map {
		vm_cont_map[vm_id] = make(map[string]float64)
		for cont_id, cont_values := range cstat_val.Containers {
			cpu_util_string := cont_values["cpu_util"]
			last_ind := len(cpu_util_string) - 1
			cpu_util_string = cpu_util_string[:last_ind]

			cpu_util_val, err := strconv.ParseFloat(cpu_util_string, 64)
			cpu_util_val = cpu_util_val / 100.0
			if err != nil {
				fmt.Println("Got Error converting CPU string to float")
			}

			vm_cont_map[vm_id][cont_id] = cpu_util_val
		}
	}

	cont_migration_map := scheduleMigration(vm_cont_map, target)

	return cont_migration_map

}

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
	cstat := VMstats[userIP]
	mu.Lock()
	cstat.AliveTime = time.Now()
	VMstats[userIP] = cstat
	// log.Println(userIP)
	mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func updateMap(data map[string]map[string]map[string]map[string]string) {
	for vm, vmmap := range data {
		log.Printf("Received message: %s :: %s", vm, vmmap)
		cstat := VMstats[vm]
		mu.Lock()
		if cstat.Containers == nil {
			cstat.Containers = make(map[string]map[string]string)
		}
		for key, element := range vmmap["value_diffs"] {
			cstat.Containers[key] = element
		}
		for key, element := range vmmap["added"] {
			cstat.Containers[key] = element
		}
		for key, _ := range vmmap["removed"] {
			delete(cstat.Containers, key)
		}
		VMstats[vm] = cstat
		mu.Unlock()
	}
}

func main() {

	VMstats = make(map[string]Cstats)
	conn, err := amqp.Dial("amqp://user1:password@0.0.0.0:5672/")
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
			log.Println("RECEIVED MESSAGE")
			var data map[string]map[string]map[string]map[string]string
			err := json.Unmarshal([]byte(d.Body), &data)
			if err != nil {
				panic(err)
			}
			updateMap(data)

			fmt.Println("-------------Map-------------")
			for k, v := range VMstats {
				fmt.Println(k, v)
				fmt.Println("--------------------------")
			}
		}
	}()

	go func() {
		for range time.Tick(time.Second * 5) {
			mu.Lock()
			migration_map := UpdateMigration(VMstats)
			mu.Unlock()

			if len(migration_map) > 0 {
				for cont_id, vm_vals := range migration_map {
					for vm_start, vm_end := range vm_vals {
						fmt.Println("Moving %s from %s ----> %s", cont_id, vm_start, vm_end)
						start_http := "http://" + vm_start + ":8080/checkpoint?id=" + cont_id
						end_http := "http://" + vm_end + ":8080/restore?id=" + cont_id

						_, err_start := http.Get(start_http)
						if err_start != nil {
							fmt.Println(err_start)
						}
						fmt.Println("Checkpointed")

						_, err_end := http.Get(end_http)
						if err_end != nil {
							fmt.Println(err_start)
						}
						fmt.Println("Restored")
					}
				}
			}
		}
	}()

	go func() {
		log.Println("Serving HeartBeat Server")
		http.HandleFunc("/heartbeat", heartbeatHandler)
		http.ListenAndServe("0.0.0.0:8090", nil)
	}()

	go func() {
		fmt.Println("-------------Map-------------")
		for k, v := range VMstats {
			fmt.Println(k, v)
			fmt.Println("--------------------------")
		}
		currtime := time.Now().Add(time.Minute * -1)
		for k, v := range VMstats {
			mu.Lock()
			if v.AliveTime.Before(currtime) {
				delete(VMstats, k)
			}
			mu.Unlock()
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
