package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

var (
	ctx                 = context.Background()
	redisClient         *redis.Client
	kafkaWriter         *kafka.Writer
	processingTopicsKey = "processingTopics"
	processedTopicsKey  = "processedTopics"
	kafkaTopic          = "media-transcoding"
	serverPort          = getEnv("PORT", "3000")
)

// Payload sent by user
type TaskPayload struct {
	TopicName string `json:"topicName"`
	ImageURL  string `json:"imageURL"`
}

func main() {
	kafkaBroker := getEnv("KAFKA_BROKER", "kafka:9092")
	redisHost := getEnv("REDIS_HOST", "redis:6379")

	// Init Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisHost,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Could not connect to Redis: %v", err)
	}
	log.Println("✅ Connected to Redis")

	// Init SQLite database
	initDatabase()
	log.Println("✅ SQLite database initialized")

	// Init Kafka writer
	kafkaWriter = kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    kafkaTopic,
		Balancer: &kafka.LeastBytes{},
	})
	log.Println("✅ Kafka writer ready")

	// Set up router
	router := mux.NewRouter()

	router.HandleFunc("/submit-topic", submitTopicHandler).Methods("POST")
	router.HandleFunc("/get-status", getStatusHandler).Methods("GET")
	router.HandleFunc("/healthz", healthCheckHandler).Methods("GET")
	router.HandleFunc("/get-super-resolution-images", getSuperResolutionImagesHandler).Methods("GET")
	router.HandleFunc("/get-recent-completed-from-redis", getRecentCompletedFromRedis)
	
	// Serve static files from ./public
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public")))

	log.Printf("🚀 Producer server running on http://localhost:%s\n", serverPort)
	log.Fatal(http.ListenAndServe(":"+serverPort, router))
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✅ Producer service is running"))
}

func submitTopicHandler(w http.ResponseWriter, r *http.Request) {
	var payload TaskPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("📨 Received task: topicName=%s, imageURL=%s\n", payload.TopicName, payload.ImageURL)

	// Insert task into SQLite database
	insertTask(payload.TopicName, payload.ImageURL)
	log.Printf("💾 Inserted task into SQLite database: topicName=%s, imageURL=%s\n", payload.TopicName, payload.ImageURL)

	messageBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ JSON marshal error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := kafkaWriter.WriteMessages(ctx, kafka.Message{Value: messageBytes}); err != nil {
		log.Printf("❌ Kafka send error: %v", err)
		http.Error(w, "Error submitting task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "✅ Task submitted successfully",
		"topicName": payload.TopicName,
		"imageURL":  payload.ImageURL,
	})

	// Save topic metadata to Redis in new flat format
	meta := map[string]string{
		"type":     "complete",
		"topic_id": payload.TopicName,
		"imageURL": payload.ImageURL,
		// "upscaledURL" will be added by the ESRGAN service
	}
	jsonMeta, err := json.Marshal(meta)
	if err != nil {
		log.Printf("❌ Failed to marshal topic metadata: %v", err)
	} else {
		key := "topic_metadata:" + payload.TopicName
		if err := redisClient.Set(ctx, key, jsonMeta, 0).Err(); err != nil {
			log.Printf("❌ Redis Set error: %v", err)
		} else {
			log.Printf("💾 Saved new flat topic metadata to Redis key: %s", key)
		}
	}
}

func getStatusHandler(w http.ResponseWriter, r *http.Request) {
	processedNames, err := redisClient.LRange(ctx, processedTopicsKey, 0, -1).Result()
	if err != nil {
		log.Printf("❌ Redis error (processed): %v", err)
		http.Error(w, "Error fetching processed tasks", http.StatusInternalServerError)
		return
	}

	processingMap, err := redisClient.HGetAll(ctx, processingTopicsKey).Result()
	if err != nil {
		log.Printf("❌ Redis error (processing): %v", err)
		http.Error(w, "Error fetching processing tasks", http.StatusInternalServerError)
		return
	}

	processedList := []map[string]string{}
	for _, topicName := range processedNames {
		key := "processed:" + topicName
		val, err := redisClient.Get(ctx, key).Result()
		if err == nil {
			var obj map[string]string
			if err := json.Unmarshal([]byte(val), &obj); err == nil {
				processedList = append(processedList, obj)
				continue
			}
		}
		// fallback if not found or not JSON
		processedList = append(processedList, map[string]string{"name": topicName})
	}

	processingList := []map[string]string{}
	for name, progress := range processingMap {
		processingList = append(processingList, map[string]string{
			"name":     name,
			"progress": progress,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"processed":  processedList,
		"processing": processingList,
	})
}

func getSuperResolutionImagesHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query(`
        SELECT topic_name, image_url, upscaled_url, created_at, completed_at
        FROM super_resolution_tasks
        WHERE status = 'completed'
        ORDER BY completed_at DESC
        LIMIT 100`)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var images []map[string]string
    for rows.Next() {
        var name, imageURL, upscaledURL string
        var createdAt, completedAt sql.NullString
        err := rows.Scan(&name, &imageURL, &upscaledURL, &createdAt, &completedAt)
        if err != nil {
            continue
        }

        images = append(images, map[string]string{
            "name":        name,
            "imageURL":    imageURL,
            "upscaledURL": upscaledURL,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(images)
}

func getRecentCompletedFromRedis(w http.ResponseWriter, r *http.Request) {
	keys, err := redisClient.Keys(ctx, "topic_metadata:*").Result()
	if err != nil {
		http.Error(w, "❌ Failed to list metadata keys from Redis", http.StatusInternalServerError)
		return
	}

	var results []map[string]string
	for _, key := range keys {
		val, err := redisClient.Get(ctx, key).Result()
		if err != nil {
			log.Printf("⚠️ Failed to get key %s: %v", key, err)
			continue
		}

		var entry map[string]string
		if err := json.Unmarshal([]byte(val), &entry); err != nil {
			log.Printf("⚠️ Failed to parse metadata from key %s: %v", key, err)
			continue
		}

		results = append(results, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}


