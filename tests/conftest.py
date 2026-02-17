import subprocess
import time

import pytest
import requests


@pytest.fixture(scope="session")
def base_url():
    """Start a minikube tunnel and yield the service URL."""
    proc = subprocess.Popen(
        [
            "minikube", "service", "gateway-svc",
            "-n", "godzilla", "--url",
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    # Read the URL from the first line of stdout
    url = proc.stdout.readline().strip()
    if not url:
        proc.kill()
        raise RuntimeError("Failed to get service URL from minikube")

    yield url

    proc.kill()
    proc.wait()


@pytest.fixture(scope="session")
def session(base_url):
    """Shared requests session with cookie persistence."""
    return requests.Session()


@pytest.fixture(scope="session")
def test_user():
    """Unique test user credentials."""
    username = f"testuser_{int(time.time())}"
    password = "testpass123"
    return {"username": username, "password": password}


@pytest.fixture(scope="session", autouse=True)
def cleanup_test_user(test_user):
    """Remove test user from MongoDB after all tests complete."""
    yield
    try:
        subprocess.run(
            [
                "kubectl", "exec", "-n", "godzilla",
                "deployment/mongodb", "--",
                "mongosh", "--quiet", "--eval",
                f"db.getSiblingDB('godzilla').users.deleteOne({{username: '{test_user['username']}'}})",
            ],
            capture_output=True,
            timeout=10,
        )
    except Exception:
        pass
