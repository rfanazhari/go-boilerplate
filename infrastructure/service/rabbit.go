package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	domainevent "github.com/rfanazhari/go-boilerplate/domain/event"
)

type RabbitPublisher struct {
	Conn     *amqp.Connection
	Channel  *amqp.Channel
	Exchange string
	Queue    string
}

type taskCompletedMessage struct {
	TaskID      string `json:"taskId"`
	CompletedAt string `json:"completedAt"`
}

func (p RabbitPublisher) PublishTaskCompleted(ctx context.Context, completed domainevent.TaskCompleted) error {
	body, err := json.Marshal(taskCompletedMessage{
		TaskID:      completed.TaskID.String(),
		CompletedAt: completed.CompletedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
	})
	if err != nil {
		return err
	}
	if err := p.Channel.PublishWithContext(ctx, p.Exchange, p.Queue, true, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	}); err != nil {
		return fmt.Errorf("publish task completed: %w", err)
	}
	return nil
}

func (p RabbitPublisher) Close() {
	if p.Channel != nil {
		_ = p.Channel.Close()
	}
	if p.Conn != nil {
		_ = p.Conn.Close()
	}
}

func DeclareTaskCompleted(ch *amqp.Channel, exchange, queue string) error {
	if err := ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	return ch.QueueBind(queue, queue, exchange, false, nil)
}
