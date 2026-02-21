import pytest
from mongomock_motor import AsyncMongoMockClient
from fastapi.testclient import TestClient

from src.main import app
from src import database


@pytest.fixture(autouse=True)
def mock_db():
    client = AsyncMongoMockClient()
    database.db = client["test_history"]

    # Also patch the db reference in the service module since it imported it at load time
    from src import service
    service.db = database.db

    yield database.db

    client.close()


@pytest.fixture()
def client():
    return TestClient(app)


AUTH_HEADER = {"X-Authenticated-User": "alice"}
