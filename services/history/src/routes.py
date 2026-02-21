from fastapi import APIRouter, Depends, HTTPException, Query, Request

from .models import (
    AddMessagesResponse,
    DiscussionResponse,
    MessageCreate,
    MessagesResponse,
)
from . import service

router = APIRouter()


def get_user(request: Request) -> str:
    user = request.headers.get("X-Authenticated-User")
    if not user:
        raise HTTPException(status_code=401, detail="Unauthorized")
    return user


@router.get("/discussion", response_model=list[DiscussionResponse])
async def get_discussions(user: str = Depends(get_user)):
    """Get discussions for the current user, sorted by most recently updated."""
    return await service.list_discussions(user)


@router.get("/discussion/{discussion_id}/messages", response_model=MessagesResponse)
async def get_messages(
    discussion_id: str,
    cursor: str | None = None,
    limit: int = Query(20, ge=1, le=100),
    _user: str = Depends(get_user),
):
    """Get messages for a discussion with cursor-based pagination (newest first)."""
    return await service.list_messages(discussion_id, cursor, limit)


@router.post("/discussion/{discussion_id}/messages", response_model=AddMessagesResponse)
async def add_messages(
    discussion_id: str,
    messages: list[MessageCreate],
    user: str = Depends(get_user),
):
    """Add a batch of messages to a discussion."""
    if not messages:
        raise HTTPException(status_code=400, detail="No messages provided")
    return await service.create_messages(
        discussion_id, user, [msg.text for msg in messages]
    )
