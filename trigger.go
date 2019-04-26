package amqp

import (
	"context"
	"fmt"
	"io/ioutil"

	"crypto/tls"
	"crypto/x509"

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
	consumerHandler []ConsumerHandler
}

type Factory struct {
}

type ConsumerHandler struct {
	conumserChannel <-chan amqp.Delivery
	handler         trigger.Handler
}
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
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
	if s.CertPem == "" || s.KeyPem == "" {
		c.conn, err = amqp.Dial(s.AmqpURI)
	} else {
		cfg := new(tls.Config)

		// see at the top
		cfg.RootCAs = x509.NewCertPool()

		if ca, err := ioutil.ReadFile(s.CertPem); err == nil {
			cfg.RootCAs.AppendCertsFromPEM(ca)
		}

		// Move the client cert and key to a location specific to your application
		// and load them here.

		if cert, err := tls.LoadX509KeyPair(s.CertPem, s.KeyPem); err == nil {
			cfg.Certificates = append(cfg.Certificates, cert)
		}

		// see a note about Common Name (CN) at the top
		c.conn, err = amqp.DialTLS(s.AmqpURI, cfg)
		if err != nil {
			return nil, err
		}

	}

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
		t.consumerHandler = append(t.consumerHandler, ConsumerHandler{conumserChannel: deliveries, handler: handler})
	}
	return nil
}

func (t *AMQPTrigger) Start() error {
	var err error
	for _, cChannel := range t.consumerHandler {

		go func() error {
			for d := range cChannel.conumserChannel {

				output := &Output{}

				output.Data = d.Body
				t.logger.Debug("The output of the queue is", output.Data)

				_, err = cChannel.handler.Handle(context.Background(), output)

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
