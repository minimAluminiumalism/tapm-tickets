package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName    = "kafka-sarama-demo"
	serviceVersion = "1.0.0"
	kafkaAddr = "127.0.0.1:9092" // kafka 地址
)

func extractHostFromBroker(broker string) string {
	if strings.Contains(broker, ":") {
		host, _, _ := strings.Cut(broker, ":")
		return host
	}
	return broker
}

func extractPortFromBroker(broker string) int {
	if strings.Contains(broker, ":") {
		_, portStr, _ := strings.Cut(broker, ":")
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}
	return 9092
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	apmToken := "xxxxxxx" // 填入 APM 控制台 token
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("xxxxxxxx"), // 填入 APM 控制台 endpoint
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithHeaders(
			map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", apmToken),
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}
 
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
 
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
 
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
 
	return tp, nil
 }

type TracedSyncProducer struct {
	producer sarama.SyncProducer
	tracer   trace.Tracer
	brokers  []string
	clientID string
}

func NewTracedSyncProducer(addrs []string, config *sarama.Config) (*TracedSyncProducer, error) {
	producer, err := sarama.NewSyncProducer(addrs, config)
	if err != nil {
		return nil, err
	}

	clientID := config.ClientID
	if clientID == "" {
		clientID = "sarama-producer"
	}

	return &TracedSyncProducer{
		producer: producer,
		tracer:   otel.GetTracerProvider().Tracer("sarama-producer"),
		brokers:  addrs,
		clientID: clientID,
	}, nil
}

func (p *TracedSyncProducer) SendMessage(ctx context.Context, msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	primaryBroker := p.brokers[0]
	brokerHost := extractHostFromBroker(primaryBroker)
	brokerPort := extractPortFromBroker(primaryBroker)

	ctx, span := p.tracer.Start(ctx, fmt.Sprintf("kafka produce %s", msg.Topic),
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingDestinationNameKey.String(msg.Topic),
			semconv.MessagingOperationKey.String("publish"),
			semconv.NetPeerNameKey.String(brokerHost),
			semconv.NetPeerPortKey.Int(brokerPort),
			attribute.String("messaging.kafka.destination.name", strings.Join(p.brokers, ",")),
			attribute.String("messaging.kafka.client_id", p.clientID),
			attribute.String("mq.broker", strings.Join(p.brokers, ",")), // 必需
			attribute.String("peer.filled", strings.Join(p.brokers, ",")), // 必需
			attribute.String("peer.service", strings.Join(p.brokers, ",")), // 必需
		),
	)
	defer span.End()

	if msg.Headers == nil {
		msg.Headers = make([]sarama.RecordHeader, 0)
	}

	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	for key, value := range carrier {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{
			Key:   []byte(key),
			Value: []byte(value),
		})
	}

	partition, offset, err = p.producer.SendMessage(msg)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("error.broker", primaryBroker),
		)
		return partition, offset, err
	}

	span.SetAttributes(
		attribute.Int64("kafka.partition", int64(partition)),
		attribute.Int64("kafka.offset", offset),
	)

	return partition, offset, nil
}

func (p *TracedSyncProducer) Close() error {
	return p.producer.Close()
}

type TracedConsumerGroupHandler struct {
	tracer   trace.Tracer
	ready    chan bool
	brokers  []string
	clientID string
	groupID  string
}

func NewTracedConsumerGroupHandler(brokers []string, clientID, groupID string) *TracedConsumerGroupHandler {
	if clientID == "" {
		clientID = "sarama-consumer"
	}
	if groupID == "" {
		groupID = "default-group"
	}

	return &TracedConsumerGroupHandler{
		tracer:   otel.GetTracerProvider().Tracer("sarama-consumer"),
		ready:    make(chan bool),
		brokers:  brokers,
		clientID: clientID,
		groupID:  groupID,
	}
}

func (h *TracedConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *TracedConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *TracedConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	primaryBroker := h.brokers[0]
	brokerHost := extractHostFromBroker(primaryBroker)
	brokerPort := extractPortFromBroker(primaryBroker)

	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			carrier := make(propagation.MapCarrier)
			for _, header := range message.Headers {
				carrier[string(header.Key)] = string(header.Value)
			}

			ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
			ctx, span := h.tracer.Start(ctx, fmt.Sprintf("kafka consume %s", message.Topic),
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					semconv.MessagingSystemKey.String("kafka"),
					semconv.MessagingDestinationNameKey.String(message.Topic),
					semconv.MessagingOperationKey.String("receive"),
					semconv.NetPeerNameKey.String(brokerHost),
					semconv.NetPeerPortKey.Int(brokerPort),
					attribute.String("messaging.kafka.destination.name", strings.Join(h.brokers, ",")),
					attribute.String("messaging.kafka.client_id", h.clientID),
					attribute.String("messaging.kafka.consumer.group", h.groupID),
					attribute.String("mq.broker", strings.Join(h.brokers, ",")), // 必需
					attribute.String("peer.filled", strings.Join(h.brokers, ",")), // 必需
					attribute.String("peer.service", strings.Join(h.brokers, ",")), // 必需
					attribute.Int64("kafka.partition", int64(message.Partition)),
					attribute.Int64("kafka.offset", message.Offset),
					attribute.String("kafka.key", string(message.Key)),
					attribute.Int("kafka.message.size", len(message.Value)),
					attribute.Int64("kafka.timestamp", message.Timestamp.Unix()),
				),
			)

			err := h.processMessage(ctx, message)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
				span.SetAttributes(
					attribute.String("error.broker", primaryBroker),
				)
				log.Printf("Error processing message: %v", err)
			} else {
				span.SetStatus(codes.Ok, "Message processed successfully")
			}

			span.End()
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *TracedConsumerGroupHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	_, span := h.tracer.Start(ctx, "process_message",
		trace.WithAttributes(
			attribute.String("message.key", string(message.Key)),
			attribute.Int("message.size", len(message.Value)),
		),
	)
	defer span.End()

	log.Printf("Processing message: topic=%s partition=%d offset=%d key=%s value=%s",
		message.Topic, message.Partition, message.Offset, string(message.Key), string(message.Value))

	time.Sleep(100 * time.Millisecond)

	return nil
}

func (h *TracedConsumerGroupHandler) Ready() <-chan bool {
	return h.ready
}

func runProducer(ctx context.Context) error {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.ClientID = "otel-producer-demo"

	producer, err := NewTracedSyncProducer([]string{kafkaAddr}, config)
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}
	defer producer.Close()

	for i := 0; i < 20; i++ {
		msg := &sarama.ProducerMessage{
			Topic: "test-topic",
			Key:   sarama.StringEncoder(fmt.Sprintf("key-%d", i)),
			Value: sarama.StringEncoder(fmt.Sprintf("Hello OpenTelemetry %d", i)),
		}

		partition, offset, err := producer.SendMessage(ctx, msg)
		if err != nil {
			log.Printf("Failed to send message: %v", err)
			continue
		}

		log.Printf("Message sent to partition %d at offset %d", partition, offset)
		time.Sleep(1 * time.Second)
	}

	return nil
}

func runConsumer(ctx context.Context) error {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.ClientID = "otel-consumer-demo"

	brokers := []string{kafkaAddr}
	groupID := "test-group"

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	defer consumerGroup.Close()

	handler := NewTracedConsumerGroupHandler(brokers, config.ClientID, groupID)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			if err := consumerGroup.Consume(ctx, []string{"test-topic"}, handler); err != nil {
				if strings.Contains(err.Error(), "context canceled") {
					return
				}
				log.Printf("Error from consumer: %v", err)
			}

			if ctx.Err() != nil {
				return
			}
		}
	}()

	<-handler.Ready()
	log.Println("Sarama consumer up and running...")

	<-ctx.Done()
	wg.Wait()

	return nil
}

func main() {
	ctx := context.Background()
	tp, err := initTracer(ctx)
	if err != nil {
		log.Fatal("Failed to initialize tracer:", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) > 1 && os.Args[1] == "consumer" {
		log.Println("Starting Kafka consumer with OpenTelemetry...")
		if err := runConsumer(ctx); err != nil {
			log.Fatal("Consumer error:", err)
		}
	} else {
		log.Println("Starting Kafka producer with OpenTelemetry...")
		if err := runProducer(ctx); err != nil {
			log.Fatal("Producer error:", err)
		}
	}
}

/*
使用方式:


运行生产者:
go run main.go

运行消费者:
go run main.go consumer

*/