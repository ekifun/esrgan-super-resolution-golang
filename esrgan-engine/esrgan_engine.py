from flask import Flask, request, jsonify
import os
import torch
import cv2
import numpy as np
import json
import redis
import RRDBNet_arch as arch
import threading
import time
import logging
import requests
from urllib.parse import urlparse
from flask import send_from_directory
from db import init_db, mark_task_completed

# ------------------ Configuration ------------------

logging.basicConfig(level=logging.INFO)
app = Flask(__name__)
PORT = int(os.getenv('PORT', 7001))

UPLOAD_DIR = os.getenv('UPLOAD_DIR', '/app/uploads')
RESULT_DIR = os.getenv('RESULT_DIR', 'results')
os.makedirs(UPLOAD_DIR, exist_ok=True)
os.makedirs(RESULT_DIR, exist_ok=True)

REDIS_HOST = os.getenv('REDIS_HOST', 'redis')
REDIS_PORT = int(os.getenv('REDIS_PORT', 6379))
PUB_SUB_CHANNEL = 'task_completed'

PROCESSING_TOPICS_KEY = "processingTopics"
PROCESSED_TOPICS_KEY = "processedTopics"
topics = {}

# ------------------ Redis Setup ------------------

try:
    redis_client = redis.Redis(
        host=REDIS_HOST,
        port=REDIS_PORT,
        db=0,
        decode_responses=True
    )
    redis_client.ping()
    logging.info("✅ Connected to Redis at %s:%d", REDIS_HOST, REDIS_PORT)
except redis.exceptions.ConnectionError as e:
    logging.error("❌ Redis connection failed: %s", str(e))
    raise

# ------------------ ESRGAN Model Setup ------------------

try:
    model_path = 'models/RRDB_ESRGAN_x4.pth'
    device = torch.device('cpu')
    model = arch.RRDBNet(3, 3, 64, 23, gc=32)
    model.load_state_dict(torch.load(model_path, map_location=device), strict=True)
    model.eval()
    model = model.to(device)
    logging.info("✅ ESRGAN model loaded from: %s", model_path)
except Exception as e:
    logging.error("❌ Failed to load ESRGAN model: %s", str(e))
    raise

# ------------------ Image Processing ------------------
def process_image(image_path, topic_id):
    logging.info(f"[{topic_id}] ▶️ Start processing image: {image_path}")
    topic = topics.get(topic_id, {})

    if not topic:
        try:
            topic_name = f"topic_{topic_id}"
            redis_topic_key = f"topic:{topic_name}"
            redis_data = redis_client.hgetall(redis_topic_key)
            if redis_data:
                topic = {
                    "topicName": redis_data.get("topicName", ""),
                    "imageURL": redis_data.get("imageURL", ""),
                    "imagePath": redis_data.get("imagePath", "")
                }
                logging.info(f"[{topic_id}] 🔁 Redis topic metadata retrieved: {topic}")
        except Exception as e:
            logging.error(f"[{topic_id}] ❌ Redis fetch error: {e}")

    topic_name = topic.get("topicName", f"topic_{topic_id}")
    image_url = topic.get("imageURL", "")
    image_path = topic.get("imagePath", image_path)

    topics[topic_id] = {
        "status": "processing",
        "progress": 0,
        "topicName": topic_name,
        "imageURL": image_url,
        "imagePath": image_path
    }

    time.sleep(2)

    if not os.path.exists(image_path):
        logging.error(f"[{topic_id}] ❌ Image path does not exist: {image_path}")
        topics[topic_id]["status"] = "failed"
        topics[topic_id]["error"] = "Image path not found."
        return

    img = cv2.imread(image_path, cv2.IMREAD_COLOR)
    if img is None:
        logging.error(f"[{topic_id}] ❌ Failed to read image: {image_path}")
        topics[topic_id]["status"] = "failed"
        topics[topic_id]["error"] = "Unable to read image."
        return

    rows, cols = 3, 6
    height, width = img.shape[:2]
    tile_h, tile_w = height // rows, width // cols
    output_img = np.zeros((height * 4, width * 4, 3), dtype=np.uint8)
    processed_tiles = 0
    total_tiles = rows * cols

    for r in range(rows):
        for c in range(cols):
            y0, y1 = r * tile_h, (r + 1) * tile_h if r < rows - 1 else height
            x0, x1 = c * tile_w, (c + 1) * tile_w if c < cols - 1 else width
            tile = img[y0:y1, x0:x1]
            tile = tile * 1.0 / 255
            tile = torch.from_numpy(np.transpose(tile[:, :, [2, 1, 0]], (2, 0, 1))).float().unsqueeze(0).to(device)

            with torch.no_grad():
                output = model(tile).data.squeeze().float().cpu().clamp_(0, 1).numpy()

            output = np.transpose(output[[2, 1, 0], :, :], (1, 2, 0))
            output = (output * 255.0).round().astype(np.uint8)

            oy0 = r * tile_h * 4
            oy1 = oy0 + output.shape[0]
            ox0 = c * tile_w * 4
            ox1 = ox0 + output.shape[1]
            output_img[oy0:oy1, ox0:ox1] = output

            processed_tiles += 1
            progress = int((processed_tiles / total_tiles) * 100)
            topics[topic_id]["progress"] = progress
            redis_client.hset("processingTopics", topic_name, progress)

            progress_event = {
                "type": "progress",
                "topic_id": topic_name,
                "progress": progress
            }
            redis_client.publish("progress_updates", json.dumps(progress_event))

            logging.info(f"[{topic_id}] 🧩 Tile {processed_tiles}/{total_tiles} processed")
            logging.info(f"[{topic_id}] 🚀 Progress {progress}% published to Redis channel 'progress_updates'")

    output_filename = f"{topic_id}_upscaled.png"
    output_path = os.path.join(RESULT_DIR, output_filename)
    cv2.imwrite(output_path, output_img)

    host = os.getenv('HOST', '13.57.143.121')
    upscaled_url = f"http://{host}:{PORT}/results/{output_filename}"

    topics[topic_id]["status"] = "completed"
    topics[topic_id]["resultPath"] = output_path
    logging.info(f"[{topic_id}] ✅ Final upscaled image saved: {output_path}")

    # ✅ Standardized complete message (SSE-compliant)
    complete_message = {
        "type": "complete",
        "topic_id": topic_name,
        "imageURL": image_url,
        "upscaledURL": upscaled_url
    }

    try:
        # Save standardized metadata in Redis
        metadata_key = f"topic_metadata:{topic_name}"
        redis_client.set(metadata_key, json.dumps(complete_message))
        logging.info(f"[{topic_id}] 💾 topic_metadata saved to Redis key: {metadata_key}")
    except Exception as e:
        logging.error(f"[{topic_id}] ❌ Failed to store topic metadata: {e}")

    try:
        redis_client.publish(PUB_SUB_CHANNEL, json.dumps(complete_message))
        logging.info(f"[{topic_id}] 📡 Completion event published to Redis channel '{PUB_SUB_CHANNEL}'")
    except Exception as e:
        logging.error(f"[{topic_id}] ❌ Failed to publish completion to Redis: {e}")

    mark_task_completed(topic_name, upscaled_url)

# ------------------ Flask Startup ------------------

@app.route('/create_topic', methods=['POST'])
def create_topic():
    data = request.json
    topic_name = data.get("topicName")
    image_url = data.get("imageURL")

    if not topic_name or not image_url:
        return jsonify({"error": "Both topicName and imageURL are required."}), 400

    parsed_url = urlparse(image_url)
    filename = os.path.basename(parsed_url.path)
    image_path = os.path.join(UPLOAD_DIR, filename)

    # Download image
    try:
        response = requests.get(image_url, timeout=10)
        if response.status_code != 200:
            return jsonify({"error": f"Failed to download image, status={response.status_code}"}), 400

        with open(image_path, 'wb') as f:
            f.write(response.content)
        logging.info(f"[{topic_name}] ✅ Image downloaded to: {image_path}")
    except Exception as e:
        logging.error(f"[{topic_name}] ❌ Failed to download image: {e}")
        return jsonify({"error": "Image download failed", "details": str(e)}), 400

    topic_id = str(len(topics) + 1)
    topics[topic_id] = {
        "status": "created",
        "topicName": topic_name,
        "imageURL": image_url,
        "imagePath": image_path
    }

    threading.Thread(target=process_image, args=(image_path, topic_id)).start()
    return jsonify({"topic_id": topic_id}), 201

@app.route('/get_topic/<topic_id>', methods=['GET'])
def get_topic(topic_id):
    return jsonify(topics.get(topic_id, {"error": "Topic not found"})), 200 if topic_id in topics else 404

@app.route('/close_topic/<topic_id>', methods=['POST'])
def close_topic(topic_id):
    if topic_id in topics:
        del topics[topic_id]
        return jsonify({"message": "Topic closed"}), 200
    return jsonify({"error": "Topic not found"}), 404

@app.route('/results/<path:filename>')
def serve_upscaled_image(filename):
    return send_from_directory(RESULT_DIR, filename)

# Run Flask app
if __name__ == '__main__':
    init_db()
    print("🚀 Starting ESRGAN Flask server...")
    app.run(host='0.0.0.0', port=PORT)
