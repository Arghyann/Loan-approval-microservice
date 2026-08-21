package main

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	log.Println("Starting Notification Service...")

	// 1. Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// 2. Open a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// 3. Declare the queue
	q, err := ch.QueueDeclare(
		"notification_queue", // name
		true,                 // durable
		false,                // delete when unused
		false,                // exclusive
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// 4. Start Consuming
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	forever := make(chan bool)

	// 5. Background loop to process messages
	go func() {
		for d := range msgs {
			log.Printf("Got a message: %s", d.Body)

			// TODO: You fill in the interesting bit!
			// Spin up a Goroutine here to send your Discord Webhook
			// go sendDiscordNotification(string(d.Body))
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// TODO: Write your sendDiscordNotification function here!
// Hint: You will need "net/http", "bytes", and "encoding/json"
