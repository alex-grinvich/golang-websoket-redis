package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/streadway/amqp"
	"log"
	"strconv"
)

var ctx = context.Background()

type ReferralPatient struct {
	Id int `json:"id"`
	Antibiotics interface{} `json:"antibiotics"`
}

func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}



func main() {
	conn, err := amqp.Dial("amqp://user:user@localhost:5672/")
	FailOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "172.20.0.5:6379",
		Password: "", // no password set

	})
	defer rdb.Close()

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	ch, err := conn.Channel()
	FailOnError(err, "Failed to open a channel")
	defer ch.Close()


	qReceiveNurse, err := ch.QueueDeclare(
		"mytpn.nurse.receive", // name
		true,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	FailOnError(err, "Failed to register a consumer")
	_, err = ch.QueueDeclare(
		"mytpn.nurse.node", // name
		true,               // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	FailOnError(err, "Failed to register a consumer")


	nurseReceive, err := ch.Consume(
		qReceiveNurse.Name, // queue
		"mytpn.nurse",     // consumer
		false,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	forever := make(chan bool)

	go func() {
		for d := range nurseReceive {

			log.Printf("Received a message: %s", d.Body)
			data := &ReferralPatient{}

			err := json.Unmarshal(d.Body,&data)

			if err != nil {
				fmt.Printf("error parse structure, %v",err)
			}




			newData := make(map[string]interface{})
			newData["antibiotics"],_ = json.Marshal(data.Antibiotics)

			key := strconv.FormatInt(int64(data.Id), 10)

			fmt.Println(key);

			err = rdb.Set(ctx,"test", d.Body, 0).Err()
			if err != nil {
				panic(err)
			}

			err = rdb.Set(ctx,"test1", "alexcc", 0).Err()
			if err != nil {
				panic(err)
			}

			//val, err := rdb.Get(ctx, key).Result()
			//if err != nil {
			//	panic(err)
			//}
			//fmt.Println(key, val)
			fmt.Printf("Send data to redis %v\n",data.Id)

			d.Ack(false)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
