package main

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "log"
)

var db *sql.DB

func initDatabase() {
    var err error
    db, err = sql.Open("sqlite3", "./super_resolution.db")
    if err != nil {
        log.Fatalf("❌ Failed to open SQLite DB: %v", err)
    }

    schema := `
    CREATE TABLE IF NOT EXISTS super_resolution_tasks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        topic_name TEXT UNIQUE NOT NULL,
        image_url TEXT NOT NULL,
        upscaled_url TEXT,
        status TEXT DEFAULT 'processing',
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        completed_at DATETIME
    );`
    if _, err := db.Exec(schema); err != nil {
        log.Fatalf("❌ Failed to create table: %v", err)
    }

    log.Println("✅ SQLite database initialized")
}

func insertTask(topicName, imageURL string) {
    _, err := db.Exec(`
        INSERT OR IGNORE INTO super_resolution_tasks (topic_name, image_url)
        VALUES (?, ?)`, topicName, imageURL)
    if err != nil {
        log.Printf("❌ Insert error: %v", err)
    } else {
        log.Printf("💾 Inserted task into SQLite: %s", topicName)
    }
}
