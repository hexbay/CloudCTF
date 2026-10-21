import os
import time
from dataclasses import dataclass
from typing import Any

import httpx
import pytest


@dataclass
class IntegrationConfig:
    base_url: str
    admin_username: str
    admin_password: str
    player_username: str
    player_password: str
    resource_sync_timeout: int
    template_build_timeout: int
    runtime_start_timeout: int
    docker_required: bool


def _env_int(name: str, default: int) -> int:
    value = os.getenv(name, "").strip()
    if not value:
        return default
    return int(value)


@pytest.fixture(scope="session")
def config() -> IntegrationConfig:
    return IntegrationConfig(
        base_url=os.getenv("CLOUDFLAG_BASE_URL", "http://127.0.0.1:8005").rstrip("/"),
        admin_username=os.getenv("CLOUDFLAG_ADMIN_USERNAME", "admin"),
        admin_password=os.getenv("CLOUDFLAG_ADMIN_PASSWORD", "admin123456"),
        player_username=os.getenv("CLOUDFLAG_PLAYER_USERNAME", "player1"),
        player_password=os.getenv("CLOUDFLAG_PLAYER_PASSWORD", "password123"),
        resource_sync_timeout=_env_int("CLOUDFLAG_RESOURCE_SYNC_TIMEOUT", 180),
        template_build_timeout=_env_int("CLOUDFLAG_TEMPLATE_BUILD_TIMEOUT", 300),
        runtime_start_timeout=_env_int("CLOUDFLAG_RUNTIME_START_TIMEOUT", 180),
        docker_required=os.getenv("CLOUDFLAG_DOCKER_REQUIRED", "true").strip().lower() != "false",
    )


class CloudFlagClient:
    def __init__(self, base_url: str) -> None:
        self.base_url = base_url
        self._client = httpx.Client(base_url=base_url, timeout=30.0, follow_redirects=True)
        self.admin_token: str | None = None
        self.player_token: str | None = None

    def close(self) -> None:
        self._client.close()

    def request(self, method: str, path: str, *, token: str | None = None, expected_status: int = 200, **kwargs: Any) -> httpx.Response:
        headers = dict(kwargs.pop("headers", {}))
        if token:
            headers["Authorization"] = f"Bearer {token}"
        response = self._client.request(method, path, headers=headers, **kwargs)
        if response.status_code != expected_status:
            raise AssertionError(
                f"{method} {path} expected {expected_status}, got {response.status_code}, body={response.text}"
            )
        return response

    def get_json(self, method: str, path: str, *, token: str | None = None, expected_status: int = 200, **kwargs: Any) -> Any:
        response = self.request(method, path, token=token, expected_status=expected_status, **kwargs)
        return response.json()

    def login_admin(self, username: str, password: str) -> str:
        response = self.request(
            "POST",
            "/api-admin/auth/login",
            data={"username": username, "password": password},
            headers={"Content-Type": "application/x-www-form-urlencoded"},
        )
        token = response.json()["access_token"]
        self.admin_token = token
        return token

    def login_player(self, username: str, password: str) -> str:
        response = self.request("POST", "/api/login", json={"username": username, "password": password})
        token = response.json()["access_token"]
        self.player_token = token
        return token

    def wait_for(self, callback, timeout: int, interval: float = 2.0, description: str = "condition") -> Any:
        deadline = time.time() + timeout
        last_error: Exception | None = None
        while time.time() < deadline:
            try:
                result = callback()
                if result:
                    return result
            except Exception as exc:  # noqa: BLE001
                last_error = exc
            time.sleep(interval)
        if last_error is not None:
            raise AssertionError(f"timeout waiting for {description}: {last_error}") from last_error
        raise AssertionError(f"timeout waiting for {description}")

    def healthcheck(self) -> dict[str, Any]:
        return self.get_json("GET", "/healthz")


@pytest.fixture(scope="session")
def client(config: IntegrationConfig) -> CloudFlagClient:
    value = CloudFlagClient(config.base_url)
    yield value
    value.close()


@dataclass
class ScenarioState:
    admin_token: str = ""
    seed_player_token: str = ""
    ci_player_token: str = ""
    team_id: int = 0
    player_id: int = 0
    ctf_competition_id: int = 0
    theory_competition_id: int = 0
    theory_question_id: int = 0
    exam_id: int = 0
    f12_template_id: int = 0
    tomcat_template_id: int = 0
    practice_challenge_id: int = 0
    competition_challenge_id: int = 0
    vuln_lab_id: int = 0


@pytest.fixture(scope="session")
def state() -> ScenarioState:
    return ScenarioState()
