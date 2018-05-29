package main

import (
	"log"
	"os"
	"net/smtp"
	"strings"
	"encoding/json"
	"github.com/streadway/amqp"
)

type EMail struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
}

func main() {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatal(err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	q, err := ch.QueueDeclare(
		"email", // name
		false,      // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		log.Fatal(err)
	}

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
		log.Fatal(err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message.. Sending Mail")

			email := &EMail{}
			e := json.Unmarshal(d.Body, email)
			if e != nil {
				log.Println(e)
				continue
			}

			auth := smtp.PlainAuth("", os.Getenv("SMTP_USER"), os.Getenv("SMTP_PASS"), strings.Split(os.Getenv("SMTP_HOST"), ":")[0])

			to := []string{email.To}
			msg := []byte("From: savood@chd.cx\r\n" +
				"To: " + email.To + "\r\n" +
				"Subject: " + email.Subject + "\r\n" +
				"\r\n" +
				email.Text +
				"\r\n")
			e = smtp.SendMail(os.Getenv("SMTP_HOST"), auth, "savood@chd.cx", to, msg)
			if e != nil {
				log.Println(e)
				continue
			}
			continue
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
