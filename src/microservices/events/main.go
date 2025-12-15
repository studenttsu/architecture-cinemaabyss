package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

type Config struct {
	Port         string
	KafkaBrokers []string
}

var (
	config        Config
	kafkaWriters  map[string]*kafka.Writer
	kafkaReaders  map[string]*kafka.Reader
	writersMutex  sync.RWMutex
	readersMutex  sync.RWMutex
)

type MovieEvent struct {
	MovieID     int      `json:"movie_id"`
	Title       string   `json:"title"`
	Action      string   `json:"action"`
	UserID      *int     `json:"user_id,omitempty"`
	Rating      *float64 `json:"rating,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Description string   `json:"description,omitempty"`
}

type UserEvent struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	Action    string `json:"action"`
	Timestamp string `json:"timestamp"`
}

type PaymentEvent struct {
	PaymentID  int     `json:"payment_id"`
	UserID     int     `json:"user_id"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
	Timestamp  string  `json:"timestamp"`
	MethodType string  `json:"method_type,omitempty"`
}

type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type EventResponse struct {
	Status    string `json:"status"`
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
	Event     Event  `json:"event"`
}

func main() {
	loadConfig()
	initKafka()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startConsumers(ctx)

	http.HandleFunc("/api/events/health", healthHandler)
	http.HandleFunc("/api/events/movie", handleMovieEvent)
	http.HandleFunc("/api/events/user", handleUserEvent)
	http.HandleFunc("/api/events/payment", handlePaymentEvent)

	port := config.Port
	if port == "" {
		port = "8082"
	}

	log.Printf("Starting Events Service on port %s", port)
	log.Printf("Kafka Brokers: %v", config.KafkaBrokers)

	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal(err)
		}
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	<-sigterm

	log.Println("Shutting down gracefully...")
	cancel()
	closeKafka()
}

func loadConfig() {
	config = Config{
		Port: os.Getenv("PORT"),
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}
	config.KafkaBrokers = strings.Split(kafkaBrokers, ",")
}

func initKafka() {
	kafkaWriters = make(map[string]*kafka.Writer)
	kafkaReaders = make(map[string]*kafka.Reader)

	topics := []string{"movie-events", "user-events", "payment-events"}

	for _, topic := range topics {
		writer := &kafka.Writer{
			Addr:         kafka.TCP(config.KafkaBrokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		}
		kafkaWriters[topic] = writer

		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        config.KafkaBrokers,
			Topic:          topic,
			GroupID:        "events-service-group",
			MinBytes:       10e3,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.LastOffset,
		})
		kafkaReaders[topic] = reader

		log.Printf("Initialized Kafka writer and reader for topic: %s", topic)
	}
}

func closeKafka() {
	writersMutex.Lock()
	defer writersMutex.Unlock()

	for topic, writer := range kafkaWriters {
		if err := writer.Close(); err != nil {
			log.Printf("Error closing writer for topic %s: %v", topic, err)
		}
	}

	readersMutex.Lock()
	defer readersMutex.Unlock()

	for topic, reader := range kafkaReaders {
		if err := reader.Close(); err != nil {
			log.Printf("Error closing reader for topic %s: %v", topic, err)
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}

func handleMovieEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var movieEvent MovieEvent
	if err := json.NewDecoder(r.Body).Decode(&movieEvent); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	event := Event{
		ID:        fmt.Sprintf("movie-%d-%s-%d", movieEvent.MovieID, movieEvent.Action, time.Now().Unix()),
		Type:      "movie",
		Timestamp: time.Now().Format(time.RFC3339),
		Payload:   movieEvent,
	}

	response, err := publishEvent("movie-events", event)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish event: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func handleUserEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var userEvent UserEvent
	if err := json.NewDecoder(r.Body).Decode(&userEvent); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	event := Event{
		ID:        fmt.Sprintf("user-%d-%s-%d", userEvent.UserID, userEvent.Action, time.Now().Unix()),
		Type:      "user",
		Timestamp: time.Now().Format(time.RFC3339),
		Payload:   userEvent,
	}

	response, err := publishEvent("user-events", event)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish event: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func handlePaymentEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var paymentEvent PaymentEvent
	if err := json.NewDecoder(r.Body).Decode(&paymentEvent); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	event := Event{
		ID:        fmt.Sprintf("payment-%d-%s-%d", paymentEvent.PaymentID, paymentEvent.Status, time.Now().Unix()),
		Type:      "payment",
		Timestamp: time.Now().Format(time.RFC3339),
		Payload:   paymentEvent,
	}

	response, err := publishEvent("payment-events", event)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish event: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func publishEvent(topic string, event Event) (*EventResponse, error) {
	writersMutex.RLock()
	writer, exists := kafkaWriters[topic]
	writersMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("writer for topic %s not found", topic)
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %v", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.ID),
		Value: eventJSON,
		Time:  time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = writer.WriteMessages(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to write message to Kafka: %v", err)
	}

	log.Printf("✓ Published event to topic '%s': ID=%s, Type=%s", topic, event.ID, event.Type)

	response := &EventResponse{
		Status:    "success",
		Partition: 0,
		Offset:    0,
		Event:     event,
	}

	return response, nil
}

func startConsumers(ctx context.Context) {
	topics := []string{"movie-events", "user-events", "payment-events"}

	for _, topic := range topics {
		go consumeEvents(ctx, topic)
	}
}

func consumeEvents(ctx context.Context, topic string) {
	readersMutex.RLock()
	reader, exists := kafkaReaders[topic]
	readersMutex.RUnlock()

	if !exists {
		log.Printf("Reader for topic %s not found", topic)
		return
	}

	log.Printf("Started consumer for topic: %s", topic)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping consumer for topic: %s", topic)
			return
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Printf("Error reading message from topic %s: %v", topic, err)
				continue
			}

			var event Event
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshaling event from topic %s: %v", topic, err)
				continue
			}

			log.Printf("✓ Consumed event from topic '%s': ID=%s, Type=%s, Partition=%d, Offset=%d",
				topic, event.ID, event.Type, msg.Partition, msg.Offset)

			processEvent(event)
		}
	}
}

func processEvent(event Event) {
	payloadJSON, _ := json.MarshalIndent(event.Payload, "", "  ")
	log.Printf("Processing event: ID=%s, Type=%s, Timestamp=%s\nPayload:\n%s",
		event.ID, event.Type, event.Timestamp, string(payloadJSON))
}
