#!/usr/bin/env -S uv run --script
# /// script
# dependencies = ["pymongo", "bcrypt", "faker"]
# ///
"""Seed MongoDB with sample users, discussions, and messages."""

import random

import bcrypt
from datetime import datetime, timedelta, timezone
from bson import ObjectId
from faker import Faker
from pymongo import MongoClient

fake = Faker()

MONGO_URI = "mongodb://localhost:27017"

client = MongoClient(MONGO_URI)

# --- Users (accounts service DB) ---

db_accounts = client["godzilla"]
users_col = db_accounts["users"]

users = []
for user in ['adri', 'camillou']:
    users.append(
        {"username": user, "password": bcrypt.hashpw(b"1234", bcrypt.gensalt()).decode()},
    )

users_col.delete_many({})
users_col.insert_many(users)
print(f"Inserted {len(users)} users (all with password: 1234)")

# --- Discussions & Messages (history service DB) ---

db_history = client["godzilla"]
discussions_col = db_history["discussions"]
messages_col = db_history["messages"]

discussions_col.delete_many({})
messages_col.delete_many({})

now = datetime.now(timezone.utc)

usernames = [u["username"] for u in users]

NUM_CONVERSATIONS = 1
MAX_MESSAGES_PER_CONVERSATION = 10
MIN_MESSAGES_PER_CONVERSATION = 3

conversations = []
for _ in range(NUM_CONVERSATIONS):
    pair = random.sample(usernames, 2)
    num_msgs = random.randint(MIN_MESSAGES_PER_CONVERSATION, MAX_MESSAGES_PER_CONVERSATION)
    offset = random.randint(10, 300)  # conversation starts 10-300 min ago
    messages = []
    for j in range(num_msgs):
        sender = pair[j % 2] if random.random() < 0.7 else pair[(j + 1) % 2]
        ts = now - timedelta(minutes=offset - j * random.randint(1, 5))
        messages.append((sender, fake.sentence(), ts))
    conversations.append({"users": pair, "messages": messages})

for conv in conversations:
    discussion_id = ObjectId()
    msgs = []
    for user_id, text, created_at in conv["messages"]:
        msgs.append({
            "discussion_id": discussion_id,
            "user_id": user_id,
            "text": text,
            "created_at": created_at,
        })

    messages_col.insert_many(msgs)

    last = msgs[-1]
    discussions_col.insert_one({
        "_id": discussion_id,
        "users": conv["users"],
        "created_at": msgs[0]["created_at"],
        "updated_at": last["created_at"],
        "last_message": {
            "message_id": str(last["_id"]),
            "user_id": last["user_id"],
            "text": last["text"],
            "created_at": last["created_at"],
        },
    })

print(f"Inserted {len(conversations)} discussions with {sum(len(c['messages']) for c in conversations)} messages")

client.close()
print("Done!")
