package amqp

import (
	"context"
	"fmt"

	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
	"github.com/streadway/amqp"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{})

func init() {
	_ = trigger.Register(&AMQPTrigger{}, &Factory{})
}

type AMQPTrigger struct {
	consumer        *Consumer
	settings        *Settings
	logger          log.Logger
	conumserChannel []<-chan amqp.Delivery
}

type Factory struct {
}

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	handler trigger.Handler
	tag     string
	done    chan error
}

// Metadata implements trigger.Factory.Metadata
func (f *Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

// New implements trigger.Factory.New
func (f *Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	s := &Settings{}
	err := metadata.MapToStruct(config.Settings, s, true)
	if err != nil {
		return nil, err
	}
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     s.ConsumerTag,
		done:    make(chan error),
	}

	c.conn, err = amqp.Dial(s.AmqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	c.channel, err = c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	return &AMQPTrigger{settings: s, consumer: c}, nil
}

func (t *AMQPTrigger) Initialize(ctx trigger.InitContext) error {

	t.logger = ctx.Logger()

	for _, handler := range ctx.GetHandlers() {

		s := &HandlerSettings{}
		err := metadata.MapToStruct(handler.Settings(), s, true)
		if err != nil {
			return err
		}
		if err = t.consumer.channel.ExchangeDeclare(
			s.Exchange,     // name of the exchange
			s.ExchangeType, // type
			true,           // durable
			false,          // delete when complete
			false,          // internal
			false,          // noWait
			nil,            // arguments
		); err != nil {
			return fmt.Errorf("Exchange Declare: %s", err)
		}
		_, err = t.consumer.channel.QueueDeclare(
			s.Queue, // name of the queue
			true,    // durable
			false,   // delete when unused
			false,   // exclusive
			false,   // noWait
			nil,     // arguments
		)
		if err != nil {
			return fmt.Errorf("Queue Declare: %s", err)
		}

		if err = t.consumer.channel.QueueBind(
			s.Queue,      // name of the queue
			s.BindingKey, // bindingKey
			s.Exchange,   // sourceExchange
			false,        // noWait
			nil,          // arguments
		); err != nil {
			return fmt.Errorf("Queue Bind: %s", err)
		}
		deliveries, err := t.consumer.channel.Consume(
			s.Queue,                // name
			t.settings.ConsumerTag, // consumerTag,
			false,                  // noAck
			false,                  // exclusive
			false,                  // noLocal
			false,                  // noWait
			nil,                    // arguments
		)
		if err != nil {
			return fmt.Errorf("Queue Consume: %s", err)
		}
		t.conumserChannel = append(t.conumserChannel, deliveries)
	}
	return nil
}

func (t *AMQPTrigger) Start() error {
	var err error
	for _, cChannel := range t.conumserChannel {
		go func() error {
			for d := range cChannel {

				output := &Output{}

				output.Data = d.Body

				_, err = t.consumer.handler.Handle(context.Background(), output)

				if err != nil {
					return err
				}

				d.Ack(false)
			}
			return nil
		}()

	}
	return nil

}

// Stop implements util.Managed.Stop
func (t *AMQPTrigger) Stop() error {
	if err := t.consumer.channel.Cancel(t.settings.ConsumerTag, true); err != nil {
		return fmt.Errorf("Consumer cancel failed: %s", err)
	}

	if err := t.consumer.conn.Close(); err != nil {
		return fmt.Errorf("AMQP connection close error: %s", err)
	}
	return nil
}
