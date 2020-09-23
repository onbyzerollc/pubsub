// THIS IS AN EXPERIMENTAL DRIVER, PLEASE USE WITH CAUTION
package kafka

import (
	"context"
	"log"
	"sync"
	"time"

	"io/ioutil"

	"github.com/gofrs/uuid"
	"github.com/jpillora/backoff"
	"github.com/onbyzerollc/pubsub"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

var (
	mutex                 = &sync.Mutex{}
	_     pubsub.Provider = &Provider{}
)

// Provider is a Kafka based pubsub provider
type Provider struct {
	writers  map[string]*kafka.Writer
	Brokers  []string
	Balancer kafka.Balancer
	Reader   *kafka.Reader
}

func (p *Provider) Shutdown() {
	p.Reader.Close()
}

// Publish publishes a message to Kafka with a uuid as the key
func (p *Provider) Publish(ctx context.Context, topic string, m *pubsub.Msg) error {
	w, err := p.writerForTopic(ctx, topic)
	if err != nil {
		return err
	}

	u1, err := uuid.NewV1()
	if err != nil {
		return err
	}

	return w.WriteMessages(ctx, kafka.Message{
		Key:   u1.Bytes(),
		Value: m.Data,
	})
}

// Subscribe implements Subscribe
func (p *Provider) Subscribe(opts pubsub.HandlerOptions, h pubsub.MsgHandler) {
	logrus.Infof("Subscribing to %s successWriter/name %s", opts.Topic, opts.ServiceName+"."+opts.Name)
	logSuccess := logrus.WithField("pubsub", "kafka")
	successWriter := logSuccess.Writer()
	p.Reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        p.Brokers,
		GroupID:        opts.ServiceName + "." + opts.Name,
		Topic:          opts.Topic,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		MaxWait:        50 * time.Millisecond,
		Logger:         log.New(successWriter, "", log.Lshortfile),
		ErrorLogger:    log.New(ioutil.Discard, "", log.Lshortfile),
	})

	b := &backoff.Backoff{
		Min:    200 * time.Millisecond,
		Max:    600 * time.Second,
		Factor: 2,
		Jitter: true,
	}

	go func() {
		for {
			ctx := context.Background()
			m, err := p.Reader.FetchMessage(ctx)
			if err != nil {
				d := b.Duration()
				logrus.Errorf(
					"Subscription receive to topic %s failed, reconnecting in %v. Err: %v",
					opts.Topic, d, err,
				)
				time.Sleep(d)
			}

			b.Reset()

			msg := pubsub.Msg{
				ID:   string(m.Key),
				Data: m.Value,
				Ack: func() {
					p.Reader.CommitMessages(ctx, m)
				},
				Nack: func() {},
			}

			err = h(ctx, msg)
			if err != nil {
				break
			}

			if opts.AutoAck {
				msg.Ack()
			}

			logrus.Debugf("message at topic/partition/offset %v/%v/%v\n",
				m.Topic, m.Partition, m.Offset)
		}
		successWriter.Close()
	}()
}

func (p *Provider) writerForTopic(ctx context.Context, topic string) (*kafka.Writer, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if p.writers == nil {
		p.writers = map[string]*kafka.Writer{}
	}

	if p.writers[topic] != nil {
		return p.writers[topic], nil
	}

	if len(p.Brokers) > 0 {
		c, err := kafka.DefaultDialer.Dial("tcp", p.Brokers[0])
		if err != nil {
			return nil, err
		}

		logrus.Debugf("Creating Topic %s in Kafka", topic)
		err = c.CreateTopics(kafka.TopicConfig{
			Topic: topic,
		})
		if err != nil {
			logrus.Errorf("Error creating Topic %s in Kafka, err %s", topic, err.Error())
			return nil, err
		}
	}

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  p.Brokers,
		Topic:    topic,
		Balancer: p.Balancer,
	})

	p.writers[topic] = w
	return w, nil
}
