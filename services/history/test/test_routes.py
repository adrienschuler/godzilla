from bson import ObjectId

from .conftest import AUTH_HEADER


def test_health(client):
    res = client.get("/health")
    assert res.status_code == 200
    assert res.json() == {"status": "healthy"}


def test_get_discussions_unauthenticated(client):
    res = client.get("/discussion")
    assert res.status_code == 401


def test_get_discussions_empty(client):
    res = client.get("/discussion", headers=AUTH_HEADER)
    assert res.status_code == 200
    assert res.json() == []


def test_post_and_get_messages(client):
    discussion_id = str(ObjectId())

    # Post messages
    res = client.post(
        f"/discussion/{discussion_id}/messages",
        json=[{"text": "hello"}, {"text": "world"}],
        headers=AUTH_HEADER,
    )
    assert res.status_code == 200
    body = res.json()
    assert body["inserted_count"] == 2
    assert body["last_message_id"]

    # Get messages back
    res = client.get(
        f"/discussion/{discussion_id}/messages",
        headers=AUTH_HEADER,
    )
    assert res.status_code == 200
    messages = res.json()["messages"]
    assert len(messages) == 2
    # Newest first
    assert messages[0]["text"] == "world"
    assert messages[1]["text"] == "hello"


def test_post_creates_discussion(client):
    discussion_id = str(ObjectId())

    client.post(
        f"/discussion/{discussion_id}/messages",
        json=[{"text": "hi"}],
        headers=AUTH_HEADER,
    )

    res = client.get("/discussion", headers=AUTH_HEADER)
    discussions = res.json()
    assert len(discussions) == 1
    assert discussion_id in discussions[0]["id"]
    assert discussions[0]["last_message"]["text"] == "hi"
    assert discussions[0]["last_message"]["user_id"] == "alice"


def test_post_empty_messages_rejected(client):
    discussion_id = str(ObjectId())
    res = client.post(
        f"/discussion/{discussion_id}/messages",
        json=[],
        headers=AUTH_HEADER,
    )
    assert res.status_code == 400


def test_cursor_pagination(client):
    discussion_id = str(ObjectId())

    # Insert 3 messages
    client.post(
        f"/discussion/{discussion_id}/messages",
        json=[{"text": "a"}, {"text": "b"}, {"text": "c"}],
        headers=AUTH_HEADER,
    )

    # Fetch with limit=2
    res = client.get(
        f"/discussion/{discussion_id}/messages",
        params={"limit": 2},
        headers=AUTH_HEADER,
    )
    body = res.json()
    assert len(body["messages"]) == 2
    assert body["next_cursor"] is not None

    # Fetch next page
    res = client.get(
        f"/discussion/{discussion_id}/messages",
        params={"limit": 2, "cursor": body["next_cursor"]},
        headers=AUTH_HEADER,
    )
    body = res.json()
    assert len(body["messages"]) == 1
    assert body["next_cursor"] is None
