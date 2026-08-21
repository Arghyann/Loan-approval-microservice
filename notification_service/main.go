package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// Attempt to load .env file (it's okay if it fails/doesn't exist)
	_ = godotenv.Load()
	
	log.Println("Starting Notification Service...")

	// 1. Connect to RabbitMQ using Environment Variable (fallback to localhost)
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	conn, err := amqp.Dial(rabbitURL)
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
		"loan_applications", // name
		true,                 // durable
		false,                // delete when unused
		false,                // exclusive
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// 4. Start Consuming with Manual ACKs
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack (SET TO FALSE!)
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

			// We pass the entire 'd' object so the goroutine can Ack/Nack it!
			go sendDiscordNotification(d)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// sendDiscordNotification sends a quick HTTP POST to your webhook
func sendDiscordNotification(d amqp.Delivery) {
	message := string(d.Body)
	discordURL := os.Getenv("DISCORD_WEBHOOK_URL")
	
	if discordURL == "" {
		log.Println("Skipping Discord notification: DISCORD_WEBHOOK_URL not set")
		d.Ack(false) // Safe to delete, it's just a missing config
		return
	}

	body, _ := json.Marshal(map[string]string{
		"content": "🚀 **New Loan Application Received:**\n```json\n" + message + "\n```",
	})

	resp, err := http.Post(discordURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Failed to send to Discord: %v", err)
		// 🚨 NETWORK ERROR! Put the message back in the queue for another worker!
		d.Nack(false, true)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Successfully pinged Discord!")
		// ✅ SUCCESS! Tell RabbitMQ to delete the message forever.
		d.Ack(false)
	} else {
		log.Printf("Discord returned status: %v", resp.StatusCode)
		// Assume bad payload or Discord is mad at us. Delete it so we don't infinitely loop.
		d.Ack(false)
	}
}
