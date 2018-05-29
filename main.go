package main

import (
	"log"
	"github.com/dgrijalva/jwt-go"
	"time"
	"os"
	"net/smtp"
	"strings"
	"github.com/globalsign/mgo/bson"
	"encoding/json"
	"github.com/streadway/amqp"
)

type User struct {
	ID    bson.ObjectId `json:"id"`
	EMail string        `json:"email"`
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
		"new_user", // name
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

			user := &User{}
			e := json.Unmarshal(d.Body, user)
			if e != nil {
				log.Println(e)
				continue
			}

			emailToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"userid": user.ID.Hex(),
				"exp":    time.Now().Add(60 * time.Minute).Unix(),
			})

			emailTokenString, e := emailToken.SignedString([]byte(os.Getenv("SECRET_EMAIL")))
			if e != nil {
				log.Println(e)
				continue
			}

			auth := smtp.PlainAuth("", os.Getenv("SMTP_USER"), os.Getenv("SMTP_PASS"), strings.Split(os.Getenv("SMTP_HOST"), ":")[0])

			to := []string{user.EMail}
			msg := []byte("From: savood@chd.cx\r\n" +
				"To: " + user.EMail + "\r\n" +
				"Subject: Bestätige deinen Savood Account!\r\n" +
				"\r\n" +
				"Bitte bestätige deinen Account: " + os.Getenv("EXTERNAL_BASE") + "/confirm?key=" + emailTokenString + "\r\n")
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
