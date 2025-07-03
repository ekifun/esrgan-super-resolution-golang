package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/go-redis/redis/v9"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
)

var (
	ctx                  = context.Background()
	kafkaTopic           = "media-transcoding"
	redisClient          *redis.Client
	pubsubClient         *redis.Client
	processingTopicsKey  = "processingTopics"
	processedTopicsKey   = "processedTopics"
	pubSubChannel        = "task_completed"
	kafkaGroupID         = "transcoding-group"
	esrganServerURL      = "http://esrgan-engine:7001"
	sseClients           = make(map[chan string]bool)
	sseClientsMutex      sync.Mutex
)

type TaskPayload struct {
	TopicName string `json:"topicName"`
	ImageURL  string `json:"imageURL"`
}

type TaskCompleteMessage struct {
	TopicID string `json:"topic_id"`
	Result  string `json:"result"`
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func main() {
	redisAddr := getEnv("REDIS_HOST", "redis:6379")
	redisClient = redis.NewClient(&redis.Options{Addr: redisAddr})
	pubsubClient = redis.NewClient(&redis.Options{Addr: redisAddr})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis connection failed: %v", err)
	}
	log.Println("✅ Connected to Redis")

	go runConsumer()
	go subscribeToTaskCompletion()

	router := mux.NewRouter()
	router.HandleFunc("/get-status", getStatusHandler).Methods("GET")
	router.HandleFunc("/events", sseHandler)

	port := getEnv("PORT", "5001")
	log.Printf("🚀 Consumer server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func runConsumer() {
	broker := getEnv("KAFKA_BROKER", "kafka:9092")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    kafkaTopic,
		GroupID:  kafkaGroupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer r.Close()
	log.Println("📥 Kafka consumer started...")

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Printf("❌ Kafka read error: %v", err)
			continue
		}
		var payload TaskPayload
		if err := json.Unmarshal(m.Value, &payload); err != nil {
			log.Printf("❌ JSON parse error: %v", err)
			continue
		}
		go processTopic(payload.TopicName, payload.ImageURL)
	}
}

func processTopic(topic, imageURL string) {
	log.Printf("🎯 Processing topic: %s, imageURL: %s", topic, imageURL)
	redisClient.HSet(ctx, processingTopicsKey, topic, "0")

	payload := TaskPayload{TopicName: topic, ImageURL: imageURL}
	jsonBytes, _ := json.Marshal(payload)
	resp, err := http.Post(fmt.Sprintf("%s/create_topic", esrganServerURL), "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		log.Printf("❌ ESRGAN request failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		log.Printf("⚠️ ESRGAN rejected task: %s", topic)
	}
}

func subscribeToTaskCompletion() {
	sub := pubsubClient.Subscribe(ctx, pubSubChannel)
	log.Printf("🔔 Subscribed to Redis Pub/Sub channel: %s", pubSubChannel)
	for {
		msg, err := sub.ReceiveMessage(ctx)
		if err != nil {
			log.Printf("❌ Pub/Sub error: %v", err)
			continue
		}
		broadcastSSE(msg.Payload)
		var complete TaskCompleteMessage
		if err := json.Unmarshal([]byte(msg.Payload), &complete); err != nil {
			log.Printf("❌ Pub/Sub message parse error: %v", err)
			continue
		}
		redisClient.HDel(ctx, processingTopicsKey, complete.TopicID)
		processed, _ := json.Marshal(complete)
		redisClient.RPush(ctx, processedTopicsKey, processed)
		log.Printf("✅ Task complete: %s, result: %s", complete.TopicID, complete.Result)
	}
}

func getStatusHandler(w http.ResponseWriter, r *http.Request) {
	processedRaw, _ := redisClient.LRange(ctx, processedTopicsKey, 0, -1).Result()
	processingMap, _ := redisClient.HGetAll(ctx, processingTopicsKey).Result()

	var processed []map[string]string
	for _, item := range processedRaw {
		var obj map[string]string
		if err := json.Unmarshal([]byte(item), &obj); err == nil {
			processed = append(processed, obj)
		}
	}

	var processing []map[string]string
	for topic, progress := range processingMap {
		processing = append(processing, map[string]string{
			"name":     topic,
			"progress": progress,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"processed":  processed,
		"processing": processing,
	})
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	messageChan := make(chan string)
	sseClientsMutex.Lock()
	sseClients[messageChan] = true
	sseClientsMutex.Unlock()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	for {
		select {
		case msg := <-messageChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ctx.Done():
			sseClientsMutex.Lock()
			delete(sseClients, messageChan)
			sseClientsMutex.Unlock()
			return
		}
	}
}

func broadcastSSE(message string) {
	sseClientsMutex.Lock()
	defer sseClientsMutex.Unlock()
	for ch := range sseClients {
		select {
		case ch <- message:
		default:
			// drop if channel is full
		}
	}
}
