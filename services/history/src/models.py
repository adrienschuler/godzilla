from pydantic import BaseModel
from typing import Optional
from datetime import datetime


class LastMessage(BaseModel):
    message_id: str
    user_id: str
    text: str
    created_at: datetime


class DiscussionResponse(BaseModel):
    id: str
    users: list[str]
    created_at: datetime
    updated_at: datetime
    last_message: Optional[LastMessage] = None


class MessageResponse(BaseModel):
    id: str
    text: str
    user_id: str
    created_at: datetime
    discussion_id: str


class MessagesResponse(BaseModel):
    messages: list[MessageResponse]
    next_cursor: Optional[str] = None


class MessageCreate(BaseModel):
    text: str


class AddMessagesResponse(BaseModel):
    inserted_count: int
    last_message_id: str
