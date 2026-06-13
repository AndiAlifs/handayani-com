import json

import httpx
import respx
from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)
GO = "http://localhost:8090"


@respx.mock
def test_login_rewrites_to_go_login():
    route = respx.post(f"{GO}/api/login").mock(
        return_value=httpx.Response(200, json={"token": "jwt123"})
    )
    r = client.post("/api/auth/login", json={"username": "a", "password": "b"})
    assert r.status_code == 200
    assert r.json() == {"token": "jwt123"}
    assert route.called
    assert json.loads(route.calls.last.request.content) == {"username": "a", "password": "b"}


@respx.mock
def test_attendance_prefix_stripped():
    route = respx.post(f"{GO}/api/clock-in").mock(
        return_value=httpx.Response(200, json={"status": "approved"})
    )
    r = client.post(
        "/api/attendance/clock-in",
        json={"latitude": 1.0, "longitude": 2.0},
        headers={"Authorization": "Bearer jwt123"},
    )
    assert r.status_code == 200
    assert route.called
    assert route.calls.last.request.headers["authorization"] == "Bearer jwt123"


@respx.mock
def test_admin_passthrough_with_query():
    route = respx.get(f"{GO}/api/admin/records").mock(
        return_value=httpx.Response(200, json=[])
    )
    r = client.get("/api/admin/records?date=2024-01-01", headers={"Authorization": "Bearer x"})
    assert r.status_code == 200
    assert route.called
    assert "date=2024-01-01" in str(route.calls.last.request.url)


@respx.mock
def test_instructor_passthrough():
    route = respx.get(f"{GO}/api/instructor/students").mock(
        return_value=httpx.Response(200, json=[])
    )
    r = client.get("/api/instructor/students", headers={"Authorization": "Bearer y"})
    assert r.status_code == 200
    assert route.called
    assert route.calls.last.request.headers["authorization"] == "Bearer y"


@respx.mock
def test_go_down_returns_502():
    respx.post(f"{GO}/api/clock-in").mock(side_effect=httpx.ConnectError("down"))
    r = client.post("/api/attendance/clock-in", json={}, headers={"Authorization": "Bearer x"})
    assert r.status_code == 502
    assert "tidak tersedia" in r.json()["error"]
