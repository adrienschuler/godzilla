from bson import ObjectId
from datetime import datetime, timezone

from .database import db
from .models import (
    AddMessagesResponse,
    DiscussionResponse,
    LastMessage,
    MessageResponse,
    MessagesResponse,
)


def _sid(obj_id: ObjectId) -> str:
    return str(obj_id)


async def list_discussions(user: str) -> list[DiscussionResponse]:
    results = []
    async for doc in db.discussions.find({"users": user}).sort("updated_at", -1):
        last_message = None
        if doc.get("last_message"):
            lm = doc["last_message"]
            last_message = LastMessage(
                message_id=_sid(lm["message_id"]),
                user_id=lm["user_id"],
                text=lm["text"],
                created_at=lm["created_at"],
            )
        results.append(
            DiscussionResponse(
                id=_sid(doc["_id"]),
                users=doc["users"],
                created_at=doc["created_at"],
                updated_at=doc["updated_at"],
                last_message=last_message,
            )
        )
    return results


async def list_messages(
    discussion_id: str, cursor: str | None, limit: int
) -> MessagesResponse:
    query = {"discussion_id": ObjectId(discussion_id)}
    if cursor:
        query["_id"] = {"$lt": ObjectId(cursor)}

    messages = []
    async for doc in db.messages.find(query).sort("_id", -1).limit(limit):
        messages.append(
            MessageResponse(
                id=_sid(doc["_id"]),
                text=doc["text"],
                user_id=doc["user_id"],
                created_at=doc["created_at"],
                discussion_id=_sid(doc["discussion_id"]),
            )
        )

    next_cursor = messages[-1].id if len(messages) == limit else None
    return MessagesResponse(messages=messages, next_cursor=next_cursor)


async def create_messages(
    discussion_id: str, user: str, texts: list[str]
) -> AddMessagesResponse:
    now = datetime.now(timezone.utc)
    docs = [
        {
            "discussion_id": ObjectId(discussion_id),
            "user_id": user,
            "text": text,
            "created_at": now,
        }
        for text in texts
    ]

    result = await db.messages.insert_many(docs)

    last_doc = docs[-1]
    last_id = result.inserted_ids[-1]

    await db.discussions.update_one(
        {"_id": ObjectId(discussion_id)},
        {
            "$set": {
                "last_message": {
                    "message_id": last_id,
                    "user_id": user,
                    "text": last_doc["text"],
                    "created_at": now,
                },
                "updated_at": now,
            },
            "$addToSet": {"users": user},
            "$setOnInsert": {"created_at": now},
        },
        upsert=True,
    )

    return AddMessagesResponse(
        inserted_count=len(texts),
        last_message_id=_sid(last_id),
    )
