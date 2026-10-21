from __future__ import annotations

"""默认 CI 种子环境下的 HTTP 集成冒烟测试。

这套测试按职责拆为四类：

- 数据准备：准备管理员登录态、用户数据、模板数据、比赛与题目数据
- 管理员测试：验证后台同步资源、编译模板、切换比赛状态等管理行为
- 用户测试：验证用户创建队伍、报名、答题、查看排行等正向链路
- 权限测试：验证不同身份与不同比赛状态下的访问边界

它只覆盖一条固定的 CI 业务主链路，不试图穷举所有部署形态。
若产品行为发生变化，应优先更新这里的前提说明和断言。
"""

import re
from datetime import datetime, timedelta, timezone
from typing import Any

import httpx
import pytest


def _find_by_name(rows: list[dict[str, Any]], name: str) -> dict[str, Any]:
    """按精确名称从接口返回列表里取出指定资源。"""
    for row in rows:
        if str(row.get("name", "")).strip() == name:
            return row
    raise AssertionError(f"row named {name!r} not found in {rows}")


def _find_repo(rows: list[dict[str, Any]], name: str) -> dict[str, Any]:
    """按仓库名定位内置资源仓库记录。"""
    for row in rows:
        if str(row.get("name", "")).strip().lower() == name.lower():
            return row
    raise AssertionError(f"resource repository {name!r} not found in {rows}")


def _iso_in(minutes: int) -> str:
    """生成若干分钟后的 UTC 时间字符串，供比赛开始/结束时间接口使用。"""
    return (datetime.now(timezone.utc) + timedelta(minutes=minutes)).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def _templates(client, token: str) -> list[dict[str, Any]]:
    """读取模板列表接口，返回当前后台可见的全部虚拟化模板。"""
    payload = client.get_json("GET", "/api-admin/virtualization/templates?current=1&pageSize=500", token=token)
    return payload["data"]


def _template_ready(client, token: str, template_id: int) -> dict[str, Any] | None:
    """轮询单个模板详情接口，直到模板构建状态变为 ready/completed。"""
    row = client.get_json("GET", f"/api-admin/virtualization/templates/{template_id}", token=token)
    if str(row.get("status", "")).lower() in {"completed", "ready"}:
        return row
    return None


def _challenge_runtime_ready(client, token: str, challenge_id: int) -> dict[str, Any] | None:
    """读取题目详情接口，确认动态题容器已经进入可访问状态。"""
    detail = client.get_json("GET", f"/api/challenges/{challenge_id}", token=token)
    container_info = detail.get("container_info")
    if container_info and str(container_info.get("status", "")).lower() in {"running", "created", "creating"}:
        return detail
    return None


def _extract_flag_from_target(detail: dict[str, Any]) -> str | None:
    """访问容器目标地址并从页面内容中提取 flag，模拟选手真实解题路径。"""
    container_info = detail.get("container_info") or {}
    targets = container_info.get("targets") or []
    if not targets:
        return None
    target_url = str(targets[0].get("url", "")).strip()
    if not target_url:
        return None
    try:
        response = httpx.get(target_url, timeout=10.0, follow_redirects=True)
        response.raise_for_status()
    except httpx.HTTPError:
        return None
    match = re.search(r"flag\{[^}]+\}", response.text)
    if not match:
        return None
    return match.group(0)


@pytest.mark.order(1)
def test_data_setup_auth_and_tokens(client, config, state) -> None:
    """数据准备：准备管理员与种子用户登录态。"""
    health = client.healthcheck()
    assert health["status"] == "ok"

    state.admin_token = client.login_admin(config.admin_username, config.admin_password)
    state.seed_player_token = client.login_player(config.player_username, config.player_password)

    admin_me = client.get_json("GET", "/api-admin/auth/me", token=state.admin_token)
    assert str(admin_me.get("username", "")).strip() == config.admin_username


@pytest.mark.order(2)
def test_data_setup_users_and_competitions(client, state) -> None:
    """数据准备：准备测试用户、比赛、理论题与试卷数据。"""
    created_player = client.get_json(
        "POST",
        "/api-admin/users",
        json={
            "username": "ci-player-01",
            "email": "ci-player-01@example.com",
            "password": "CloudFlag123!",
            "role_id": 3,
        },
        token=state.admin_token,
    )
    state.player_id = int(created_player["id"])

    state.ci_player_token = client.login_player("ci-player-01", "CloudFlag123!")

    ctf_competition = client.get_json(
        "POST",
        "/api-admin/competitions",
        json={
            "name": "CI CTF Competition",
            "type": "ctf",
            "participant_mode": "team",
            "status": "draft",
            "is_public": True,
            "registration_open": True,
            "start_time": _iso_in(60),
            "end_time": _iso_in(180),
        },
        token=state.admin_token,
    )
    state.ctf_competition_id = int(ctf_competition["id"])

    theory_competition = client.get_json(
        "POST",
        "/api-admin/competitions",
        json={
            "name": "CI Theory Competition",
            "type": "theory",
            "participant_mode": "individual",
            "status": "draft",
            "is_public": True,
            "registration_open": True,
            "start_time": _iso_in(60),
            "end_time": _iso_in(180),
        },
        token=state.admin_token,
    )
    state.theory_competition_id = int(theory_competition["id"])

    theory_question = client.get_json(
        "POST",
        "/api-admin/theory/questions",
        json={
            "question": "CI Theory Question: What is XSS?",
            "points": 10,
            "question_type": "text",
            "context": {"answer": "XSS", "options": []},
        },
        token=state.admin_token,
    )
    state.theory_question_id = int(theory_question["id"])

    client.get_json(
        "POST",
        f"/api-admin/theory/competitions/{state.theory_competition_id}/questions",
        json={"question_id": state.theory_question_id},
        token=state.admin_token,
    )

    exam = client.get_json(
        "POST",
        f"/api-admin/exam/competitions/{state.theory_competition_id}/exam",
        json={"title": "CI Theory Paper", "question_ids": [state.theory_question_id], "total_score": 10, "is_active": True},
        token=state.admin_token,
    )
    state.exam_id = int(exam["id"])


@pytest.mark.order(3)
def test_admin_sync_builtin_repositories(client, config, state) -> None:
    """管理员测试：同步内置资源仓库并确认模板已落库。"""
    repos_payload = client.get_json("GET", "/api-admin/resource-repositories", token=state.admin_token)
    repo_rows = repos_payload["data"] if isinstance(repos_payload, dict) and "data" in repos_payload else repos_payload
    ctfdb_repo = _find_repo(repo_rows, "CTFDB")
    vulhub_repo = _find_repo(repo_rows, "Vulhub")

    client.get_json("POST", f"/api-admin/resource-repositories/{ctfdb_repo['id']}/sync", token=state.admin_token)
    client.get_json("POST", f"/api-admin/resource-repositories/{vulhub_repo['id']}/sync", token=state.admin_token)

    def wait_templates_synced() -> list[dict[str, Any]] | None:
        rows = _templates(client, state.admin_token)
        if any(str(row.get("name", "")).lower() == "f12" for row in rows) and any(
            "tomcat" in str(row.get("name", "")).lower() for row in rows
        ):
            return rows
        return None

    rows = client.wait_for(
        wait_templates_synced,
        timeout=config.resource_sync_timeout,
        description="F12 and tomcat templates synced",
    )
    state.f12_template_id = int(_find_by_name(rows, "f12")["id"])
    state.tomcat_template_id = int(next(row for row in rows if "tomcat" in str(row.get("name", "")).lower())["id"])


@pytest.mark.order(4)
def test_admin_build_templates(client, config, state) -> None:
    """管理员测试：编译动态题与靶场模板。"""
    client.get_json("POST", f"/api-admin/virtualization/templates/{state.f12_template_id}/build", token=state.admin_token)
    client.get_json("POST", f"/api-admin/virtualization/templates/{state.tomcat_template_id}/build", token=state.admin_token)

    f12_row = client.wait_for(
        lambda: _template_ready(client, state.admin_token, state.f12_template_id),
        timeout=config.template_build_timeout,
        description="F12 template build",
    )
    tomcat_row = client.wait_for(
        lambda: _template_ready(client, state.admin_token, state.tomcat_template_id),
        timeout=config.template_build_timeout,
        description="tomcat template build",
    )
    assert int(f12_row["id"]) == state.f12_template_id
    assert int(tomcat_row["id"]) == state.tomcat_template_id


@pytest.mark.order(5)
def test_data_setup_challenges(client, state) -> None:
    """数据准备：基于模板准备练习题、比赛题。"""
    f12_template = client.get_json("GET", f"/api-admin/virtualization/templates/{state.f12_template_id}", token=state.admin_token)
    host_id = f12_template.get("host_id") or 1

    practice_challenge = client.get_json(
        "POST",
        "/api-admin/challenges",
        json={
            "name": "CI F12 Practice Challenge",
            "category": "web",
            "points": 100,
            "flag": "flag{ci-practice}",
            "is_dynamic": True,
            "is_public": True,
            "is_active": True,
            "usage": "PRACTICE",
            "template_id": state.f12_template_id,
            "host_id": host_id,
        },
        token=state.admin_token,
    )
    state.practice_challenge_id = int(practice_challenge["id"])

    competition_challenge = client.get_json(
        "POST",
        "/api-admin/challenges",
        json={
            "name": "CI F12 Competition Challenge",
            "category": "web",
            "points": 100,
            "flag": "flag{ci-competition}",
            "is_dynamic": True,
            "is_public": True,
            "is_active": True,
            "usage": "COMPETITION",
            "template_id": state.f12_template_id,
            "host_id": host_id,
            "competition_id": state.ctf_competition_id,
        },
        token=state.admin_token,
    )
    state.competition_challenge_id = int(competition_challenge["id"])


@pytest.mark.order(6)
def test_user_create_team_and_register_competitions(client, state) -> None:
    """用户测试：用户创建队伍，并用前台接口完成团队赛/个人赛报名。"""
    created_team = client.get_json(
        "POST",
        "/api/team",
        json={"name": "ci-team-01", "description": "CI integration team"},
        token=state.ci_player_token,
    )
    state.team_id = int(created_team["id"])

    my_team = client.get_json("GET", "/api/team/my", token=state.ci_player_token)
    assert int(my_team["id"]) == state.team_id
    assert str(my_team["name"]).strip() == "ci-team-01"

    registered_team = client.get_json(
        "POST",
        f"/api/competitions/{state.ctf_competition_id}/register",
        json={"team_id": state.team_id},
        token=state.ci_player_token,
    )
    assert registered_team["success"] is True

    registered_individual = client.get_json(
        "POST",
        f"/api/competitions/{state.theory_competition_id}/register",
        token=state.ci_player_token,
    )
    assert registered_individual["success"] is True


@pytest.mark.order(7)
def test_permission_competition_access_before_start(client, state) -> None:
    """权限测试：比赛未开始时，用户不能访问比赛题和理论赛内容。"""
    anonymous_start = client.request(
        "POST",
        f"/api/challenges/{state.practice_challenge_id}/start",
        expected_status=401,
    )
    assert "auth required" in anonymous_start.text

    pre_submit = client.request(
        "POST",
        f"/api/challenges/{state.competition_challenge_id}/submit",
        token=state.ci_player_token,
        json={"flag": "flag{ci-competition}"},
        expected_status=400,
    )
    assert "active container" in pre_submit.text or "challenge is not available" in pre_submit.text

    pre_exam = client.request(
        "GET",
        f"/api/exam/{state.theory_competition_id}",
        token=state.ci_player_token,
        expected_status=403,
    )
    assert "exam is not available" in pre_exam.text
    assert "questions" not in pre_exam.text


@pytest.mark.order(8)
def test_user_practice_challenge_flow(client, state) -> None:
    """用户测试：用户启动练习题、访问靶机并提交 flag。"""
    practice_started = client.get_json("POST", f"/api/challenges/{state.practice_challenge_id}/start", token=state.ci_player_token)
    assert practice_started["success"] is True

    practice_detail = client.wait_for(
        lambda: _challenge_runtime_ready(client, state.ci_player_token, state.practice_challenge_id),
        timeout=120,
        description="practice challenge runtime start",
    )
    assert practice_detail["container_info"] is not None

    practice_flag = client.wait_for(
        lambda: _extract_flag_from_target(practice_detail),
        timeout=60,
        description="practice challenge flag extraction",
    )
    practice_submit = client.get_json(
        "POST",
        f"/api/challenges/{state.practice_challenge_id}/submit",
        token=state.ci_player_token,
        json={"flag": practice_flag},
    )
    assert practice_submit["correct"] is True

    practice_board = client.get_json("GET", "/api/leaderboard/practice", token=state.ci_player_token)
    assert int(practice_board["total"]) >= 1
    assert any(int(row.get("user_id", 0)) == state.player_id for row in practice_board["data"])


@pytest.mark.order(9)
def test_admin_start_competitions(client, state) -> None:
    """管理员测试：切换比赛状态到 ongoing，开放正式比赛访问。"""
    ctf_status = client.get_json(
        "PATCH",
        f"/api-admin/competitions/{state.ctf_competition_id}/status",
        json={"status": "ongoing"},
        token=state.admin_token,
    )
    theory_status = client.get_json(
        "PATCH",
        f"/api-admin/competitions/{state.theory_competition_id}/status",
        json={"status": "ongoing"},
        token=state.admin_token,
    )
    assert str(ctf_status.get("status", "")).strip() == "ongoing"
    assert str(theory_status.get("status", "")).strip() == "ongoing"


@pytest.mark.order(10)
def test_permission_no_admin_data_for_player(client, state) -> None:
    """权限测试：普通用户不能访问后台管理接口。"""
    admin_me = client.request("GET", "/api-admin/auth/me", token=state.ci_player_token, expected_status=403)
    assert "admin permissions required" in admin_me.text

    admin_templates = client.request(
        "GET",
        "/api-admin/virtualization/templates?current=1&pageSize=10",
        token=state.ci_player_token,
        expected_status=403,
    )
    assert "admin permissions required" in admin_templates.text


@pytest.mark.order(11)
def test_permission_only_registered_users_can_access_competition_resources(client, state) -> None:
    """权限测试：未报名用户不能访问比赛题和理论赛。"""
    outsider_start = client.request(
        "POST",
        f"/api/challenges/{state.competition_challenge_id}/start",
        token=state.seed_player_token,
        expected_status=403,
    )
    assert (
        "challenge is not available" in outsider_start.text
        or "participant" in outsider_start.text
        or "user is not in a team" in outsider_start.text
    )

    outsider_exam = client.request(
        "GET",
        f"/api/exam/{state.theory_competition_id}",
        token=state.seed_player_token,
        expected_status=403,
    )
    assert "competition is not available" in outsider_exam.text or "exam is not available" in outsider_exam.text


@pytest.mark.order(12)
def test_user_competition_challenge_exam_and_scoreboard_flow(client, state) -> None:
    """用户测试：用户启动比赛题、提交考试，并验证比赛排行与个人排行。"""
    started = client.get_json("POST", f"/api/challenges/{state.competition_challenge_id}/start", token=state.ci_player_token)
    assert started["success"] is True

    competition_detail = client.wait_for(
        lambda: _challenge_runtime_ready(client, state.ci_player_token, state.competition_challenge_id),
        timeout=120,
        description="competition challenge runtime start",
    )
    competition_flag = client.wait_for(
        lambda: _extract_flag_from_target(competition_detail),
        timeout=60,
        description="competition challenge flag extraction",
    )
    post_submit = client.get_json(
        "POST",
        f"/api/challenges/{state.competition_challenge_id}/submit",
        token=state.ci_player_token,
        json={"flag": competition_flag},
    )
    assert post_submit["correct"] is True

    exam_detail = client.get_json("GET", f"/api/exam/{state.theory_competition_id}", token=state.ci_player_token)
    assert exam_detail["success"] is True
    assert "questions" in exam_detail
    if exam_detail["questions"]:
        first_question = exam_detail["questions"][0]
        context = first_question.get("context") or {}
        assert "answer" not in context

    exam_submit = client.get_json(
        "POST",
        f"/api/exam/{state.theory_competition_id}/submit",
        token=state.ci_player_token,
        json={"exam_id": state.exam_id, "answers": {str(state.theory_question_id): "XSS"}},
    )
    assert exam_submit["success"] is True
    assert int(exam_submit["exam_id"]) == state.exam_id

    competition_board = client.get_json(
        "GET",
        f"/api/competitions/{state.ctf_competition_id}/scoreboard",
        token=state.ci_player_token,
    )
    assert int(competition_board["total"]) >= 1
    assert any(
        int(row.get("user_id", 0)) == state.player_id
        or int(row.get("team_id", 0)) == state.team_id
        for row in competition_board["data"]
    )

    my_rank = client.get_json("GET", "/api/leaderboard/my-rank", token=state.ci_player_token)
    assert int(my_rank["user_id"]) == state.player_id
    assert int(my_rank["rank"]) >= 1

    user_stats = client.get_json("GET", "/api/user/stats", token=state.ci_player_token)
    assert str(user_stats.get("team_name", "")).strip() == "ci-team-01"


@pytest.mark.order(13)
def test_user_vuln_lab_runtime_flow(client, config, state) -> None:
    """用户测试：用户启动 Vuln Lab 并确认运行时已经拉起。"""
    tomcat_template = client.get_json("GET", f"/api-admin/virtualization/templates/{state.tomcat_template_id}", token=state.admin_token)
    if not tomcat_template.get("host_id") and config.docker_required:
        pytest.skip("docker host is not configured for tomcat template")

    vuln_lab = client.get_json(
        "POST",
        "/api-admin/vuln-labs",
        json={
            "name": "CI Tomcat Vuln Lab",
            "cve_id": "CI-TOMCAT",
            "template_id": state.tomcat_template_id,
            "is_public": True,
            "is_active": True,
            "timeout_seconds": 1800,
        },
        token=state.admin_token,
    )
    state.vuln_lab_id = int(vuln_lab["id"])

    runtime = client.get_json("POST", f"/api/vuln-labs/{state.vuln_lab_id}/start", token=state.ci_player_token)
    assert runtime["success"] is True

    def wait_vuln_lab_running() -> dict[str, Any] | None:
        detail = client.get_json("GET", f"/api/vuln-labs/{state.vuln_lab_id}", token=state.ci_player_token)
        runtime_data = detail.get("runtime")
        if runtime_data and str(runtime_data.get("status", "")).lower() in {"running", "created", "creating"}:
            return detail
        return None

    detail = client.wait_for(
        wait_vuln_lab_running,
        timeout=config.runtime_start_timeout,
        description="vuln lab runtime start",
    )
    assert detail["runtime"] is not None
