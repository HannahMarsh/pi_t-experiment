package queue

import (
	//"github.com/streadway/amqp"

	"github.com/HannahMarsh/PrettyLogger"
	amqp "github.com/rabbitmq/amqp091-go"
)

func ConnectToQueue() (*amqp.Connection, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, PrettyLogger.WrapError(err, "queue.ConnectToQueue(): error connecting to queue")
	}
	return conn, nil
}

func PublishMessage(conn *amqp.Connection, queueName string, message []byte) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
