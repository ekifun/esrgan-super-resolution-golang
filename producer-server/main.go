package main

import (
    "encoding/json"
    "log"
    "net/http"

    "github.com/gorilla/mux"
    "github.com/redis/go-redis/v9"
    "github.com/segmentio/kafka-go"
)


var (
	ctx                  = context.Background()
	redisClient          *redis.Client
	kafkaWriter          *kafka.Writer
	processingTopicsKey  = "processingTopics"
	processedTopicsKey   = "processedTopics"
	kafkaTopic           = "media-transcoding"
	kafkaBroker          = "kafka:9092"
	redisHost            = "redis:6379"
	serverPort           = "3000"
)

// Payload sent by user
type TaskPayload struct {
	TopicName string `json:"topicName"`
	ImageURL  string `json:"imageURL"`
}

func main() {
	// Init Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisHost,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Could not connect to Redis: %v", err)
	}
	log.Println("✅ Connected to Redis")

	// Init Kafka writer
	kafkaWriter = kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    kafkaTopic,
		Balancer: &kafka.LeastBytes{},
	})
	log.Println("✅ Kafka writer ready")

	// Set up router
	router := mux.NewRouter()
	// Serve static files from the /public directory
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public")))
	router.HandleFunc("/submit-topic", submitTopicHandler).Methods("POST")
	router.HandleFunc("/get-status", getStatusHandler).Methods("GET")

	log.Printf("🚀 Producer server running on http://localhost:%s\n", serverPort)
	log.Fatal(http.ListenAndServe(":"+serverPort, router))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Producer service is running."))
}

func submitTopicHandler(w http.ResponseWriter, r *http.Request) {
	var payload TaskPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("API to accept task: topicName=%s, imageURL=%s\n", payload.TopicName, payload.ImageURL)

	messageBytes, _ := json.Marshal(payload)
	err := kafkaWriter.WriteMessages(ctx, kafka.Message{Value: messageBytes})
	if err != nil {
		log.Printf("❌ Kafka send error: %v", err)
		http.Error(w, "Error submitting task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Task submitted successfully",
		"topicName": payload.TopicName,
		"imageURL":  payload.ImageURL,
	})
}

func getStatusHandler(w http.ResponseWriter, r *http.Request) {
	processed, err := redisClient.LRange(ctx, processedTopicsKey, 0, -1).Result()
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
	for _, topicStr := range processed {
		var obj map[string]string
		if err := json.Unmarshal([]byte(topicStr), &obj); err == nil {
			processedList = append(processedList, obj)
		} else {
			processedList = append(processedList, map[string]string{"name": topicStr})
		}
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
