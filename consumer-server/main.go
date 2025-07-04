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
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

var (
	ctx                 = context.Background()
	kafkaTopic          = "media-transcoding"
	redisClient         *redis.Client
	pubsubClient        *redis.Client
	processingTopicsKey = "processingTopics"
	processedTopicsKey  = "processedTopics"
	pubSubChannel       = "task_completed"
	progressChannel     = "progress_updates"
	kafkaGroupID        = "transcoding-group"
	esrganServerURL     = "http://esrgan-engine:7001"

	sseClients      = make(map[chan string]bool)
	sseClientsMutex sync.Mutex
)

type TaskPayload struct {
	TopicName string `json:"topicName"`
	ImageURL  string `json:"imageURL"`
}

type TaskCompleteMessage struct {
	Type        string `json:"type"`
	TopicID     string `json:"topic_id"`
	ImageURL    string `json:"imageURL"`
	UpscaledURL string `json:"upscaledURL"`
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
	go subscribeToProgressUpdates()

	router := mux.NewRouter()
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

	redisClient.Set(ctx, "imageURL:"+topic, imageURL, 0)
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
		log.Printf("✅ Task complete message: %s", msg.Payload)

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Payload), &raw); err != nil {
			log.Printf("❌ Pub/Sub message parse error: %v", err)
			continue
		}

		topicID, _ := raw["topic_id"].(string)
		result, _ := raw["upscaledURL"].(string)
		if topicID == "" || result == "" {
			log.Printf("⚠️ Skipping incomplete completion message: %+v", raw)
			continue
		}

		redisClient.HDel(ctx, processingTopicsKey, topicID)

		imageURL, err := redisClient.Get(ctx, "imageURL:"+topicID).Result()
		if err != nil {
			log.Printf("⚠️ Could not find imageURL for topic %s: %v", topicID, err)
			imageURL = ""
		}

		completeMessage := map[string]string{
			"type":        "complete",
			"topic_id":    topicID,
			"imageURL":    imageURL,
			"upscaledURL": result,
		}

		jsonMeta, err := json.Marshal(completeMessage)
		if err != nil {
			log.Printf("❌ Failed to marshal metadata: %v", err)
			continue
		}

		log.Printf("📤 Broadcasting SSE: %s", jsonMeta)
		broadcastSSE(string(jsonMeta))
	}
}

func subscribeToProgressUpdates() {
	progressSub := pubsubClient.Subscribe(ctx, progressChannel)
	log.Printf("🔁 Subscribed to Redis Pub/Sub channel: %s", progressChannel)

	for {
		msg, err := progressSub.ReceiveMessage(ctx)
		if err != nil {
			log.Printf("❌ Progress Pub/Sub error: %v", err)
			continue
		}
		log.Printf("📊 Progress update: %s", msg.Payload)
		broadcastSSE(msg.Payload)
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://13.57.143.121:3000")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	messageChan := make(chan string)
	sseClientsMutex.Lock()
	sseClients[messageChan] = true
	sseClientsMutex.Unlock()

	log.Println("📡 SSE client connected")
	fmt.Fprintf(w, "data: %s\n\n", `{"type":"info","message":"connected"}`)
	flusher.Flush()

	ctx := r.Context()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-messageChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-ctx.Done():
			log.Println("❌ SSE client disconnected")
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
			log.Printf("📤 SSE broadcasted: %s", message)
		default:
			log.Println("⚠️ Dropped SSE message due to full channel")
		}
	}
}



