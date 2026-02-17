"""End-to-end integration tests for the godzilla auth system.

These tests run against a live minikube cluster and validate the full
user lifecycle: register -> login -> logout. Edge cases (missing fields,
etc.) are covered by the Ruby unit tests in services/accounts/test/.
"""

import pytest


class TestAuthFlow:
    def test_register_new_user(self, base_url, session, test_user):
        resp = session.post(
            f"{base_url}/user/register",
            json=test_user,
        )
        assert resp.status_code == 201

    def test_register_duplicate_user(self, base_url, session, test_user):
        resp = session.post(
            f"{base_url}/user/register",
            json=test_user,
        )
        assert resp.status_code == 409

    def test_login_success_sets_cookie(self, base_url, session, test_user):
        resp = session.post(
            f"{base_url}/user/login",
            json=test_user,
        )
        assert resp.status_code == 200
        assert resp.json()["status"] == "success"
        assert "auth_token" in session.cookies.keys()

    def test_logout_clears_cookie(self, base_url, session, test_user):
        if "auth_token" not in session.cookies.keys():
            resp = session.post(f"{base_url}/user/login", json=test_user)
            assert resp.status_code == 200

        resp = session.post(f"{base_url}/user/logout")
        assert resp.status_code == 200
        assert resp.json()["status"] == "success"
        assert "auth_token" not in session.cookies.keys()

    def test_login_invalid_credentials(self, base_url, session):
        resp = session.post(
            f"{base_url}/user/login",
            json={"username": "invalid_user", "password": "wrong_password"},
        )
        assert resp.status_code == 401
        assert "auth_token" not in session.cookies.keys()
