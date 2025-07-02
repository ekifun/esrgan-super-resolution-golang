import sqlite3
import os
import logging

DB_PATH = os.getenv("DB_PATH", "/app/data/super_resolution.db")

def init_db():
    try:
        with sqlite3.connect(DB_PATH) as conn:
            conn.execute('''
                CREATE TABLE IF NOT EXISTS super_resolution_tasks (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    topic_name TEXT UNIQUE NOT NULL,
                    image_url TEXT NOT NULL,
                    upscaled_url TEXT,
                    status TEXT DEFAULT 'processing',
                    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                    completed_at DATETIME
                );
            ''')
            conn.commit()
        logging.info("✅ SQLite database initialized")
    except Exception as e:
        logging.error(f"❌ Failed to init DB: {e}")

def mark_task_completed(topic_name, upscaled_url):
    try:
        with sqlite3.connect(DB_PATH) as conn:
            conn.execute('''
                UPDATE super_resolution_tasks
                SET upscaled_url = ?, status = 'completed', completed_at = CURRENT_TIMESTAMP
                WHERE topic_name = ?
            ''', (upscaled_url, topic_name))
            conn.commit()
        logging.info(f"[{topic_name}] ✅ SQLite updated with result")
    except Exception as e:
        logging.error(f"[{topic_name}] ❌ SQLite update failed: {e}")
