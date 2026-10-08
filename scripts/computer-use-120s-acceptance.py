#!/usr/bin/env python3
"""Run the real desktop Computer Use 120-second acceptance loop.

This wrapper only drives the authenticated local session-control API. It never
launches or manipulates a desktop application itself: target launch, observe,
input, wait, and stop must all be model-issued ComputerUse actions routed by
the desktop-v2 host.

The wrapper deliberately keeps credentials in memory only. Evidence contains
provider/model and redacted action metadata, but never the desktop auth token,
provider API keys, or raw request headers.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import dataclasses
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import struct
import sys
import time
import zlib
import urllib.error
import urllib.request
import uuid
from typing import Any, Callable, Iterable, Mapping, Optional


DEFAULT_PROVIDER = "jiuan-responses-gpt-5.6sol"
DEFAULT_MODEL = "gpt-6-sol"
DEFAULT_EFFORT = "high"
DEFAULT_PERMISSION_MODE = "allow"
DEFAULT_PROMPT_MODE = "code"
DEFAULT_TOTAL_BUDGET_SECONDS = 120.0
DEFAULT_OPERATIONAL_DEADLINE_SECONDS = 110.0
DEFAULT_POLL_INTERVAL_SECONDS = 0.5
DEFAULT_WORKSPACE = Path(__file__).resolve().parents[1]
DEFAULT_APP_PATH = DEFAULT_WORKSPACE / "desktop-v2/build/bin/go-e2e.app"
DEFAULT_OUTPUT = DEFAULT_WORKSPACE / "desktop-v2/build/validation" / dt.datetime.now().strftime("%Y%m%d") / "computer-use-gpt6-sol-120s"

TERMINAL_STATUSES = {"completed", "failed", "stopped", "idle", "archived"}
COMPUTER_TOOL_NAME = "ComputerUse"
TARGET_ID = "workbuddy"
TARGET_BUNDLE_ID = "com.workbuddy.workbuddy"
SELF_BUNDLE_IDS = {"com.wails.go-e2e", "com.openai.codex"}
EXPECTED_TYPED_TEXT = "1+1=2"
PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"


class AcceptanceError(RuntimeError):
    """A validation failure with a user-visible stage."""

    def __init__(self, stage: str, message: str):
        super().__init__(message)
        self.stage = stage


@dataclasses.dataclass(frozen=True)
class LocalServer:
    desktop_pid: int
    server_pid: int
    port: int
    token: str
    started_at: str = ""


@dataclasses.dataclass
class Timing:
    started_at: str
    start_monotonic: float
    launch_started: Optional[float] = None
    launch_finished: Optional[float] = None
    binding_started: Optional[float] = None
    binding_finished: Optional[float] = None
    readiness_finished: Optional[float] = None
    click_duration: float = 0.0
    type_duration: float = 0.0
    send_duration: float = 0.0
    reply_wait_duration: float = 0.0
    stop_duration: float = 0.0

    def duration(self, started: Optional[float], finished: Optional[float]) -> float:
        if started is None or finished is None:
            return 0.0
        return round(max(0.0, finished - started), 3)


class Evidence:
    """Write private, JSON-only evidence without ever persisting credentials."""

    def __init__(self, root: Path):
        self.root = root
        if self.root.exists() and any(self.root.iterdir()):
            raise AcceptanceError("evidence_isolation", "evidence directory is not empty; use a fresh run directory")
        self.root.mkdir(parents=True, exist_ok=True, mode=0o700)
        try:
            os.chmod(self.root, 0o700)
        except OSError:
            pass

    def json(self, name: str, value: Any) -> None:
        path = self.root / name
        path.write_text(json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        try:
            os.chmod(path, 0o600)
        except OSError:
            pass

    def text(self, name: str, value: str) -> None:
        path = self.root / name
        path.write_text(value, encoding="utf-8")
        try:
            os.chmod(path, 0o600)
        except OSError:
            pass

    def bytes(self, name: str, value: bytes) -> Path:
        path = self.root / name
        path.write_bytes(value)
        try:
            os.chmod(path, 0o600)
        except OSError:
            pass
        return path


class APIClient:
    """Authenticated local HTTP client; auth is never included in evidence."""

    def __init__(self, port: int, token: str, timeout: float = 10.0):
        self.base = f"http://127.0.0.1:{port}"
        self.token = token
        self.timeout = timeout

    def _headers(self, mutation: bool = False) -> dict[str, str]:
        headers = {
            "Authorization": f"Bearer {self.token}",
            "X-Tenant-Key": "webui-local",
            "X-User-Id": "webui-local-user",
            "Content-Type": "application/json",
        }
        if mutation:
            headers["Idempotency-Key"] = "computer-use-120s-" + uuid.uuid4().hex
        return headers

    def request(self, path: str, method: str = "GET", body: Optional[Mapping[str, Any]] = None, timeout: Optional[float] = None) -> Any:
        payload = None if body is None else json.dumps(body, ensure_ascii=False).encode("utf-8")
        request = urllib.request.Request(
            self.base + path,
            method=method,
            headers=self._headers(mutation=body is not None),
            data=payload,
        )
        try:
            with urllib.request.urlopen(request, timeout=timeout or self.timeout) as response:
                raw = response.read()
        except urllib.error.HTTPError as exc:
            detail = exc.read(256).decode("utf-8", errors="replace")
            raise RuntimeError(f"HTTP {exc.code} for {method} {path}: {detail}") from exc
        except (urllib.error.URLError, TimeoutError, OSError) as exc:
            raise RuntimeError(f"request failed for {method} {path}: {exc}") from exc
        try:
            return json.loads(raw.decode("utf-8"))
        except json.JSONDecodeError as exc:
            raise RuntimeError(f"invalid JSON response for {method} {path}") from exc

    def download(self, path: str, timeout: Optional[float] = None) -> tuple[bytes, str]:
        request = urllib.request.Request(self.base + path, headers=self._headers())
        try:
            with urllib.request.urlopen(request, timeout=timeout or self.timeout) as response:
                return response.read(), response.headers.get("Content-Type", "")
        except urllib.error.HTTPError as exc:
            detail = exc.read(256).decode("utf-8", errors="replace")
            raise RuntimeError(f"HTTP {exc.code} for GET {path}: {detail}") from exc
        except (urllib.error.URLError, TimeoutError, OSError) as exc:
            raise RuntimeError(f"download failed for GET {path}: {exc}") from exc


def unwrap(value: Any) -> Any:
    if isinstance(value, dict) and "data" in value:
        return value["data"]
    return value


def iso_now() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat()


def parse_iso(value: str) -> Optional[float]:
    if not value:
        return None
    try:
        return dt.datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp()
    except (TypeError, ValueError):
        return None


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def sha256_text(value: str) -> str:
    return sha256_bytes(value.encode("utf-8"))


def get_status(session_payload: Any) -> tuple[dict[str, Any], str]:
    value = unwrap(session_payload)
    if isinstance(value, dict) and isinstance(value.get("session"), dict):
        value = value["session"]
    if not isinstance(value, dict):
        raise RuntimeError("session response has no session object")
    return value, str(value.get("status", ""))


def find_local_server(workspace: Path = DEFAULT_WORKSPACE) -> LocalServer:
    """Find the newest desktop-v2 server child without printing its token."""

    rows = subprocess.check_output(["ps", "-axo", "pid=,ppid=,command="], text=True).splitlines()
    desktops: list[tuple[int, int, str]] = []
    for row in rows:
        fields = row.split(None, 2)
        if len(fields) != 3:
            continue
        pid, ppid, command = fields
        if command.endswith("/Contents/MacOS/go-e2e-desktop") and str(workspace) + "/" in command:
            desktops.append((int(pid), int(ppid), command))
    if not desktops:
        raise RuntimeError("running desktop-v2 app was not found")

    desktop_pid = max(desktops, key=lambda item: item[0])[0]
    servers: list[tuple[int, str]] = []
    for row in rows:
        fields = row.split(None, 2)
        if len(fields) != 3:
            continue
        pid, ppid, command = fields
        if int(ppid) != desktop_pid or "--desktop-local" not in command:
            continue
        if "/desktop-v2/" not in command or " server " not in f" {command} ":
            continue
        servers.append((int(pid), command))
    if not servers:
        raise RuntimeError("desktop-v2 local server child was not found")
    if len(servers) > 1:
        servers.sort(key=lambda item: item[0])
    server_pid, command = servers[-1]
    args = shlex.split(command)
    try:
        port = int(args[args.index("--port") + 1])
        token = args[args.index("--auth-token") + 1]
    except (ValueError, IndexError) as exc:
        raise RuntimeError("desktop-v2 local server arguments are incomplete") from exc
    return LocalServer(desktop_pid=desktop_pid, server_pid=server_pid, port=port, token=token)


def source_commit(workspace: Path) -> str:
    try:
        return subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=workspace, text=True, stderr=subprocess.DEVNULL).strip()
    except (OSError, subprocess.CalledProcessError):
        return ""


def sha256_file(path: Path) -> str:
    sha = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            sha.update(chunk)
    return sha.hexdigest()


def executable_build_id(path: Path) -> str:
    return subprocess.check_output(["go", "tool", "buildid", str(path)], text=True, stderr=subprocess.DEVNULL).strip()


def app_identity(app_path: Path) -> dict[str, Any]:
    result: dict[str, Any] = {"path": str(app_path), "exists": app_path.exists()}
    if app_path.exists():
        result["modified_at"] = dt.datetime.fromtimestamp(app_path.stat().st_mtime, tz=dt.timezone.utc).isoformat()
        manifest_path = app_path / "Contents/Resources/computer-use-build.json"
        if manifest_path.exists():
            result["manifest"] = json.loads(manifest_path.read_text())
            result["desktop_sha256"] = sha256_file(app_path / "Contents/MacOS/go-e2e-desktop")
    return result


def validate_build_identity(workspace: Path, app_path: Path) -> None:
    identity = app_identity(app_path)
    manifest = identity.get("manifest", {})
    if not manifest or manifest.get("source_commit") != source_commit(workspace) or manifest.get("source_dirty") is not False:
        raise AcceptanceError("build_identity", "desktop app is not an attested clean build of current HEAD; rebuild via scripts/build-desktop-v2.sh")
    expected_build_id = manifest.get("desktop_build_id")
    if not expected_build_id or expected_build_id != executable_build_id(app_path / "Contents/MacOS/go-e2e-desktop"):
        raise AcceptanceError("build_identity", "desktop executable does not match the source build manifest")
    if manifest.get("helper_sha256") != sha256_file(app_path / "Contents/Helpers/ComputerHelper.app/Contents/MacOS/computer-helper-macos"):
        raise AcceptanceError("build_identity", "helper bytes do not match the source build manifest")



def json_from_text(value: Any) -> Optional[dict[str, Any]]:
    if not isinstance(value, str):
        return value if isinstance(value, dict) else None
    try:
        parsed = json.loads(value)
    except json.JSONDecodeError:
        return None
    return parsed if isinstance(parsed, dict) else None


def regex_value(value: str, key: str) -> str:
    match = re.search(rf'"{re.escape(key)}"\s*:\s*"([^"]+)"', value)
    return match.group(1) if match else ""


def regex_number(value: str, key: str) -> Optional[int]:
    match = re.search(rf'"{re.escape(key)}"\s*:\s*(\d+)', value)
    return int(match.group(1)) if match else None


def compact_frame(value: Any) -> Optional[dict[str, float]]:
    """Keep only the non-sensitive window geometry from a tool result."""
    if not isinstance(value, dict):
        return None
    frame: dict[str, float] = {}
    for key in ("x", "y", "width", "height"):
        number = value.get(key)
        if isinstance(number, (int, float)) and not isinstance(number, bool):
            frame[key] = float(number) if isinstance(number, float) else number
    return frame if len(frame) == 4 else None


def merge_window_identity(item: dict[str, Any], value: Any) -> None:
    """Extract stable target-window identity from a known response object."""
    if not isinstance(value, dict):
        return
    if not item.get("window_id") and value.get("id") not in (None, ""):
        item["window_id"] = str(value["id"])
    if not item.get("bundle_id") and value.get("bundle_id") not in (None, ""):
        item["bundle_id"] = str(value["bundle_id"])
    owner_pid = value.get("owner_pid")
    if item.get("owner_pid") is None and isinstance(owner_pid, int) and not isinstance(owner_pid, bool):
        item["owner_pid"] = max(0, owner_pid)
    frame = compact_frame(value.get("frame"))
    if frame and "window_frame" not in item:
        item["window_frame"] = frame


def merge_tool_identity(item: dict[str, Any], output: Any) -> None:
    """Decode the stable fields in both legacy and atomic-observe responses."""
    if not isinstance(output, dict):
        return
    for key in ("window_id", "bundle_id", "target_id", "outcome", "error_code"):
        value = output.get(key)
        if value not in (None, "") and key not in item:
            item[key] = str(value)
    merge_window_identity(item, output.get("target_window"))
    merge_window_identity(item, output.get("active_window"))
    launch = output.get("launch_receipt")
    if isinstance(launch, dict):
        for key in ("target_id", "bundle_id", "outcome", "error_code"):
            value = launch.get(key)
            if value not in (None, "") and key not in item:
                item[key] = str(value)
        merge_window_identity(item, launch.get("window"))
    observation = output.get("observation")
    if isinstance(observation, dict):
        merge_window_identity(item, observation.get("active_window"))
        capabilities = observation.get("capabilities")
        if isinstance(capabilities, dict):
            merge_window_identity(item, capabilities.get("target_window"))


def event_payload(event: Mapping[str, Any]) -> dict[str, Any]:
    raw = event.get("payload_json", "")
    if isinstance(raw, dict):
        return dict(raw)
    if not isinstance(raw, str):
        return {}
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        return {}
    return parsed if isinstance(parsed, dict) else {}


def action_from_event(event: Mapping[str, Any]) -> str:
    payload = event_payload(event)
    raw = payload.get("input")
    parsed = json_from_text(raw)
    if isinstance(parsed, dict):
        return str(parsed.get("action", ""))
    return ""


def compact_observation(value: Any) -> Optional[dict[str, Any]]:
    if not isinstance(value, dict):
        return None
    allowed = ("observation_id", "asset_id", "media_type", "name", "size_bytes", "sha256")
    result = {key: value[key] for key in allowed if key in value}
    return result or None


def event_time(event: Mapping[str, Any]) -> Optional[float]:
    return parse_iso(str(event.get("created_at", "")))


def tool_events(events: Iterable[Mapping[str, Any]]) -> list[dict[str, Any]]:
    calls: dict[str, dict[str, Any]] = {}
    results: dict[str, dict[str, Any]] = {}
    ordered: list[dict[str, Any]] = []
    for event in events:
        payload = event_payload(event)
        tool_id = str(payload.get("tool_id", ""))
        if event.get("event_type") == "tool_call" and payload.get("tool_name") == COMPUTER_TOOL_NAME:
            calls[tool_id] = dict(event)
        elif event.get("event_type") == "tool_result" and payload.get("tool_name") == COMPUTER_TOOL_NAME:
            results[tool_id] = dict(event)
    for tool_id, call in calls.items():
        result = results.get(tool_id)
        if result is None:
            continue
        call_payload = event_payload(call)
        result_payload = event_payload(result)
        output = result_payload.get("output", "")
        if not isinstance(output, str):
            output = json.dumps(output, ensure_ascii=False)
        observation = compact_observation(result_payload.get("computer_observation"))
        action = action_from_event(call)
        call_at = event_time(call)
        result_at = event_time(result)
        item: dict[str, Any] = {
            "tool_id": tool_id,
            "action": action,
            "call_at": call.get("created_at", ""),
            "result_at": result.get("created_at", ""),
            "is_error": bool(result_payload.get("is_error", False)),
            "output_sha256": sha256_text(output),
            "output_length": len(output),
        }
        if call_at is not None and result_at is not None:
            item["duration_seconds"] = round(max(0.0, result_at - call_at), 3)
        if observation:
            item["computer_observation"] = observation
        # Output is deliberately bounded by the server. Extract only safe
        # identity fields from it; never persist the output itself. The atomic
        # first-observe response nests the launch window under
        # launch_receipt.window, so parse that known schema instead of relying
        # on a flat regex match.
        merge_tool_identity(item, result_payload)
        decoded_output = json_from_text(output)
        if decoded_output is not None:
            merge_tool_identity(item, decoded_output)
        else:
            for key in ("window_id", "bundle_id", "target_id", "outcome", "error_code"):
                value = regex_value(output, key)
                if value:
                    item[key] = value
        if action == "observe" and ('"launch_receipt"' in output or isinstance(result_payload.get("launch_receipt"), dict)):
            # The generic first-observe fast path may return launch, binding,
            # and the target observation as one host-authorized result.
            item["implicit_launch"] = True
        owner_pid = regex_number(output, "owner_pid")
        if owner_pid is not None:
            item["owner_pid"] = owner_pid
        # The redacted tool input carries only counts. Keep them for evidence.
        redacted = json_from_text(call_payload.get("input"))
        if isinstance(redacted, dict):
            for key in ("input_bytes", "text_length", "key_length", "keys_count"):
                if key in redacted and isinstance(redacted[key], int):
                    item[key] = max(0, redacted[key])
        ordered.append(item)
    ordered.sort(key=lambda item: item.get("call_at", ""))
    return ordered


def validate_run_actions(actions: Iterable[Mapping[str, Any]], started_at: str, ended_at: str) -> None:
    """Fail closed on stale receipts, missing timestamps or reversed events."""
    started, ended = parse_iso(started_at), parse_iso(ended_at)
    if started is None or ended is None or ended < started:
        raise AcceptanceError("evidence_isolation", "invalid run time boundary")
    for action in actions:
        call, result = parse_iso(str(action.get("call_at", ""))), parse_iso(str(action.get("result_at", "")))
        if call is None or result is None or call < started or result < call or result > ended:
            raise AcceptanceError("evidence_isolation", "ComputerUse receipt timestamps do not belong to the current run")


def png_has_visible_content(data: bytes) -> bool:
    """Return false for a structurally valid all-white screenshot.

    This is intentionally a small, dependency-free blank-screen guard. It does
    not OCR text or contain WorkBuddy selectors; the saved PNG remains the
    authoritative visual evidence for human inspection.
    """
    if not data.startswith(PNG_SIGNATURE):
        return False
    pos = len(PNG_SIGNATURE)
    width = height = bit_depth = color_type = 0
    idat = bytearray()
    while pos + 12 <= len(data):
        length = struct.unpack(">I", data[pos : pos + 4])[0]
        chunk_type = data[pos + 4 : pos + 8]
        chunk = data[pos + 8 : pos + 8 + length]
        pos += 12 + length
        if chunk_type == b"IHDR" and len(chunk) >= 13:
            width, height, bit_depth, color_type = struct.unpack(">IIBB", chunk[:10])
        elif chunk_type == b"IDAT":
            idat.extend(chunk)
        elif chunk_type == b"IEND":
            break
    if not width or not height or bit_depth != 8 or color_type not in (2, 6) or not idat:
        return True
    channels = 3 if color_type == 2 else 4
    stride = width * channels
    try:
        raw = zlib.decompress(bytes(idat))
    except zlib.error:
        return True
    if len(raw) < height * (stride + 1):
        return True
    previous = bytearray(stride)
    sample_step = max(1, width // 64)
    for row in range(height):
        start = row * (stride + 1)
        filter_type = raw[start]
        encoded = raw[start + 1 : start + 1 + stride]
        current = bytearray(encoded)
        for index in range(stride):
            left = current[index - channels] if index >= channels else 0
            up = previous[index]
            upper_left = previous[index - channels] if index >= channels else 0
            if filter_type == 1:
                current[index] = (current[index] + left) & 0xFF
            elif filter_type == 2:
                current[index] = (current[index] + up) & 0xFF
            elif filter_type == 3:
                current[index] = (current[index] + ((left + up) // 2)) & 0xFF
            elif filter_type == 4:
                estimate = left + up - upper_left
                pa, pb, pc = abs(estimate - left), abs(estimate - up), abs(estimate - upper_left)
                predictor = left if pa <= pb and pa <= pc else up if pb <= pc else upper_left
                current[index] = (current[index] + predictor) & 0xFF
            elif filter_type != 0:
                return True
        for x in range(0, width, sample_step):
            pixel = current[x * channels : (x + 1) * channels]
            if len(pixel) >= 3 and any(channel < 250 for channel in pixel[:3]):
                return True
        previous = current
    return False


def validate_png(data: bytes, media_type: str) -> None:
    if media_type.split(";", 1)[0].lower() != "image/png" or not data.startswith(PNG_SIGNATURE) or len(data) < 32:
        raise ValueError("observation asset is not a valid PNG")
    if not png_has_visible_content(data):
        raise ValueError("observation screenshot is blank/white")


def save_observation_assets(
    client: APIClient, evidence: Evidence, actions: list[dict[str, Any]], deadline: float
) -> tuple[dict[str, str], list[dict[str, Any]]]:
    """Download screenshots without allowing one bad asset to hide the trace.

    Conversation events and redacted action receipts are persisted before this
    function runs. Each observation is therefore independently classified as
    saved or invalid; callers can still validate the action sequence and report
    the exact screenshot failure that prevented closure.
    """
    paths: dict[str, str] = {}
    asset_rows: list[dict[str, Any]] = []
    seen: set[str] = set()
    number = 0
    for action in actions:
        observation = action.get("computer_observation")
        if not isinstance(observation, dict):
            continue
        asset_id = str(observation.get("asset_id", ""))
        observation_id = str(observation.get("observation_id", ""))
        if not asset_id or not observation_id or observation_id in seen:
            continue
        seen.add(observation_id)
        row: dict[str, Any] = {
            "observation_id": observation_id,
            "asset_id": asset_id,
            "media_type": str(observation.get("media_type", "")),
        }
        remaining = deadline - time.monotonic()
        if remaining <= 0.25:
            row.update({"status": "invalid", "error": "evidence_deadline_expired"})
            asset_rows.append(row)
            continue
        try:
            data, media_type = client.download(
                f"/tenant/media/assets/{asset_id}", timeout=min(8.0, remaining - 0.1)
            )
            validate_png(data, media_type or str(observation.get("media_type", "")))
            number += 1
            name = f"observation-{number:02d}-{observation_id}.png"
            path = evidence.bytes(name, data)
            paths[observation_id] = str(path)
            row.update({"status": "saved", "path": str(path), "bytes": len(data)})
        except Exception as exc:
            row.update({"status": "invalid", "error": str(exc)[:240]})
        asset_rows.append(row)
    evidence.json("observation-assets.json", {
        "schema_version": "computer-use-observation-assets.v1",
        "saved_count": sum(1 for row in asset_rows if row.get("status") == "saved"),
        "invalid_count": sum(1 for row in asset_rows if row.get("status") == "invalid"),
        "assets": asset_rows,
    })
    return paths, [row for row in asset_rows if row.get("status") == "invalid"]


def safe_event_trace(events: Iterable[Mapping[str, Any]], actions: list[dict[str, Any]]) -> dict[str, Any]:
    action_by_tool = {item["tool_id"]: item for item in actions}
    output: list[dict[str, Any]] = []
    for event in events:
        payload = event_payload(event)
        item: dict[str, Any] = {
            "id": event.get("id"),
            "event_type": event.get("event_type"),
            "created_at": event.get("created_at"),
            "task_id": event.get("task_id"),
            "trace_id": event.get("trace_id"),
            "source": event.get("source"),
            "surface": event.get("surface"),
            "channel": event.get("channel"),
        }
        tool_id = str(payload.get("tool_id", ""))
        if tool_id in action_by_tool:
            item["tool_id"] = tool_id
            item["tool_name"] = COMPUTER_TOOL_NAME
            item["action"] = action_by_tool[tool_id].get("action", "")
            item["action_receipt_sha256"] = action_by_tool[tool_id].get("output_sha256", "")
            item["action_receipt_length"] = action_by_tool[tool_id].get("output_length", 0)
            if action_by_tool[tool_id].get("computer_observation"):
                item["computer_observation"] = action_by_tool[tool_id]["computer_observation"]
        elif event.get("event_type") in {"message", "thinking_delta", "usage", "message_stop"}:
            if event.get("event_type") == "message":
                content = payload.get("content", "")
                item["content_length"] = len(content) if isinstance(content, str) else 0
            elif event.get("event_type") == "thinking_delta":
                content = payload.get("content", "")
                item["content_length"] = len(content) if isinstance(content, str) else 0
            elif event.get("event_type") == "message_stop":
                item["stop_reason"] = payload.get("stop_reason", "")
        output.append(item)
    return {"schema_version": "computer-use-conversation-trace.v1", "events": output}


def execution_log_metrics(events: Iterable[Mapping[str, Any]], actions: list[dict[str, Any]]) -> dict[str, Any]:
    """Compute elapsed gaps from persisted session events, not wrapper guesses."""
    previous_result: Optional[float] = None
    action_rows: list[dict[str, Any]] = []
    native_seconds = 0.0
    between_action_seconds = 0.0
    for item in actions:
        call_at = parse_iso(str(item.get("call_at", "")))
        result_at = parse_iso(str(item.get("result_at", "")))
        if call_at is None or result_at is None:
            continue
        native = max(0.0, result_at - call_at)
        gap = None if previous_result is None else max(0.0, call_at - previous_result)
        native_seconds += native
        if gap is not None:
            between_action_seconds += gap
        action_rows.append({
            "action": item.get("action", ""),
            "call_at": item.get("call_at", ""),
            "result_at": item.get("result_at", ""),
            "gap_after_previous_result_seconds": round(gap, 3) if gap is not None else None,
            "native_action_duration_seconds": round(native, 3),
        })
        previous_result = result_at
    usage_rows: list[dict[str, Any]] = []
    message_stops: list[dict[str, Any]] = []
    for event in events:
        payload = event_payload(event)
        if event.get("event_type") == "usage":
            usage_rows.append({
                "turn": payload.get("turn"),
                "input_tokens": payload.get("input_tokens"),
                "output_tokens": payload.get("output_tokens"),
                "cache_read_input_tokens": payload.get("cache_read_input_tokens"),
                "total_tokens": payload.get("total_tokens"),
                "created_at": event.get("created_at", ""),
            })
        elif event.get("event_type") == "message_stop":
            message_stops.append({
                "turn": payload.get("turn"),
                "stop_reason": payload.get("stop_reason", ""),
                "created_at": event.get("created_at", ""),
            })
    return {
        "action_rows": action_rows,
        "action_count": len(action_rows),
        "native_action_seconds": round(native_seconds, 3),
        "between_action_seconds": round(between_action_seconds, 3),
        "usage_rows": usage_rows,
        "message_stops": message_stops,
    }


def latest_conversation_events(client: APIClient, path: str, timeout: float = 8.0) -> list[dict[str, Any]]:
    page = unwrap(client.request(path + "/conversation", timeout=timeout))
    if not isinstance(page, dict):
        return []
    events = page.get("events", [])
    return [event for event in events if isinstance(event, dict)]


def validate_effective_config(session: Mapping[str, Any], provider: str, model: str, effort: str) -> dict[str, Any]:
    effective = {
        "effective_provider": str(session.get("provider", "")),
        "effective_model": str(session.get("model", "")),
        "effective_effort": str(session.get("effort", "")),
        "status": str(session.get("status", "")),
    }
    if effective["effective_provider"] != provider or effective["effective_model"] != model:
        raise AcceptanceError(
            "effective_config",
            f"effective provider/model mismatch: provider={effective['effective_provider']!r} model={effective['effective_model']!r}",
        )
    if effective["effective_effort"] != effort:
        raise AcceptanceError("effective_config", f"effective effort mismatch: {effective['effective_effort']!r}")
    return effective


def validate_actions(
    actions: list[dict[str, Any]],
    output_paths: Mapping[str, str],
    final_status: str,
    launch_target: str = TARGET_ID,
    invalid_assets: Optional[list[Mapping[str, Any]]] = None,
) -> dict[str, Any]:
    if not actions:
        raise AcceptanceError("conversation_trace", "no ComputerUse actions were persisted")
    action_names = [str(item.get("action", "")) for item in actions]
    if action_names[0] != "observe":
        raise AcceptanceError("conversation_trace", "the first ComputerUse action was not observe")
    launch_indices = [
        index for index, item in enumerate(actions)
        if item.get("action") == "launch_app" or item.get("implicit_launch")
    ]
    if not launch_indices:
        raise AcceptanceError("launch", "no target launch receipt was persisted")
    launch_index = launch_indices[0]
    launch = actions[launch_index]
    implicit_launch = bool(launch.get("implicit_launch"))
    if launch.get("is_error") or launch.get("outcome") not in {"executed", ""}:
        raise AcceptanceError("launch", "launch receipt was not successful")
    launch_bundle = str(launch.get("bundle_id", ""))
    launch_window = str(launch.get("window_id", ""))
    if launch.get("target_id") not in {None, "", launch_target}:
        raise AcceptanceError("launch", "launch receipt target_id did not match the requested target")
    if launch_bundle and launch_bundle != TARGET_BUNDLE_ID:
        raise AcceptanceError("window binding", f"launch bound unexpected bundle {launch_bundle!r}")
    if not launch_window:
        raise AcceptanceError("window binding", "launch receipt did not include a window_id")

    post_launch = actions[launch_index + 1 :]
    if not implicit_launch and not any(item.get("action") == "observe" for item in post_launch):
        raise AcceptanceError("content readiness", "no target-window observation followed launch_app")
    if implicit_launch and not launch.get("computer_observation"):
        raise AcceptanceError("content readiness", "atomic launch/bind result had no target observation")
    type_items = [item for item in post_launch if item.get("action") == "type"]
    if len(type_items) != 1:
        raise AcceptanceError("type", f"expected exactly one type action, got {len(type_items)}")
    if type_items[0].get("text_length") != len(EXPECTED_TYPED_TEXT):
        raise AcceptanceError("type", "type action length did not match the acceptance input")
    click_items = [item for item in post_launch if item.get("action") == "click"]
    if not click_items:
        raise AcceptanceError("click", "no click action was persisted")
    wait_items = [item for item in post_launch if item.get("action") == "wait"]
    if not wait_items:
        raise AcceptanceError("reply wait", "no wait action was persisted")
    stop_items = [item for item in post_launch if item.get("action") == "stop"]
    if len(stop_items) != 1 or stop_items[0].get("is_error"):
        raise AcceptanceError("stop", "successful ComputerUse stop receipt was not persisted")
    if final_status not in TERMINAL_STATUSES:
        raise AcceptanceError("stop", f"session status is not terminal: {final_status!r}")

    for item in post_launch:
        if item.get("is_error"):
            raise AcceptanceError(str(item.get("action") or "action"), f"ComputerUse {item.get('action')} returned an error")
        action = str(item.get("action", ""))
        if action in {"observe", "click", "type", "key", "hotkey", "scroll", "drag", "double_click", "right_click", "move", "wait"}:
            window_id = str(item.get("window_id", ""))
            bundle_id = str(item.get("bundle_id", ""))
            if window_id and window_id != launch_window:
                raise AcceptanceError("window binding", f"{action} used window_id {window_id!r}, expected {launch_window!r}")
            if bundle_id and bundle_id != TARGET_BUNDLE_ID:
                raise AcceptanceError("window binding", f"{action} targeted bundle {bundle_id!r}")
            if bundle_id in SELF_BUNDLE_IDS:
                raise AcceptanceError("window binding", f"{action} targeted the control application")
    invalid_ids = [str(item.get("observation_id", "")) for item in (invalid_assets or []) if item.get("observation_id")]
    invalid_detail = f"; invalid observations: {', '.join(invalid_ids)}" if invalid_ids else ""
    if not output_paths:
        raise AcceptanceError("content readiness", f"no screenshot evidence was downloaded{invalid_detail}")
    reply_candidates = [item for item in wait_items if item.get("computer_observation", {}).get("observation_id") in output_paths]
    if not reply_candidates:
        raise AcceptanceError("reply wait", f"wait did not produce a saved reply observation screenshot{invalid_detail}")
    return {
        "target_id": launch_target,
        "launch_mode": "atomic_observe" if implicit_launch else "launch_app",
        "bundle_id": launch_bundle or TARGET_BUNDLE_ID,
        "window_id": launch_window,
        "action_count": len(actions),
        "actions": action_names,
        "reply_observation_id": reply_candidates[-1]["computer_observation"]["observation_id"],
        "screenshot_count": len(output_paths),
    }


def validate_observe_only(actions: list[dict[str, Any]], output_paths: Mapping[str, str], final_status: str) -> dict[str, Any]:
    if not actions or actions[0].get("action") != "observe":
        raise AcceptanceError("observe_only", "first action must be target observe")
    if any(item.get("action") not in {"observe", "launch_app", "wait", "stop"} for item in actions):
        raise AcceptanceError("observe_only", "observe-only run must not post input")
    launch = next((item for item in actions if item.get("implicit_launch") or item.get("action") == "launch_app"), {})
    if not launch or launch.get("is_error") or launch.get("outcome") != "executed" or launch.get("bundle_id") != TARGET_BUNDLE_ID or not launch.get("window_id"):
        raise AcceptanceError("launch", "observe-only launch/binding was not successful")
    if actions[-1].get("action") != "stop" or actions[-1].get("is_error") or final_status not in TERMINAL_STATUSES:
        raise AcceptanceError("stop", "observe-only run did not stop")
    for item in actions[:-1]:
        if item.get("is_error"):
            raise AcceptanceError("observe_only", "observe-only native operation failed")
        if item.get("window_id") and item["window_id"] != launch["window_id"]:
            raise AcceptanceError("window binding", "observe-only capture changed target identity")
    if not output_paths:
        raise AcceptanceError("content readiness", "observe-only has no valid saved observation")
    return {"target_id": TARGET_ID, "bundle_id": TARGET_BUNDLE_ID, "window_id": launch["window_id"],
            "action_count": len(actions), "screenshot_count": len(output_paths), "mode": "observe_only"}


def build_timing(timing: Timing, final_status: str, end_monotonic: float, provider: str, model: str, effort: str, error_stage: str = "") -> dict[str, Any]:
    total = max(0.0, end_monotonic - timing.start_monotonic)
    return {
        "provider": provider,
        "model": model,
        "effort": effort,
        "start_time": timing.started_at,
        "end_time": iso_now(),
        "total_elapsed_seconds": round(total, 3),
        "launch_duration": timing.duration(timing.launch_started, timing.launch_finished),
        "window_binding_duration": timing.duration(timing.binding_started, timing.binding_finished),
        "readiness_duration": timing.duration(timing.launch_finished, timing.readiness_finished),
        "click_duration": round(timing.click_duration, 3),
        "type_duration": round(timing.type_duration, 3),
        "send_duration": round(timing.send_duration, 3),
        "reply_wait_duration": round(timing.reply_wait_duration, 3),
        "stop_duration": round(timing.stop_duration, 3),
        "final_status": final_status,
        "failure_stage": error_stage,
        "within_120_seconds": total <= DEFAULT_TOTAL_BUDGET_SECONDS,
    }


def run_acceptance(
    *,
    client: APIClient,
    evidence: Evidence,
    workspace: Path = DEFAULT_WORKSPACE,
    app_path: Path = DEFAULT_APP_PATH,
    provider: str = DEFAULT_PROVIDER,
    model: str = DEFAULT_MODEL,
    effort: str = DEFAULT_EFFORT,
    total_budget: float = DEFAULT_TOTAL_BUDGET_SECONDS,
    operational_deadline: float = DEFAULT_OPERATIONAL_DEADLINE_SECONDS,
    poll_interval: float = DEFAULT_POLL_INTERVAL_SECONDS,
    observe_only: bool = False,
    clock: Callable[[], float] = time.monotonic,
    sleep: Callable[[float], None] = time.sleep,
) -> dict[str, Any]:
    start = clock()
    timing = Timing(started_at=iso_now(), start_monotonic=start)
    deadline = start + total_budget
    operational = min(deadline, start + operational_deadline)
    session_path = ""
    status_history: list[dict[str, Any]] = []
    final_status = "failed"
    error_stage = ""
    log_metrics: dict[str, Any] = {}
    message_future: Optional[concurrent.futures.Future[Any]] = None
    executor = concurrent.futures.ThreadPoolExecutor(max_workers=1, thread_name_prefix="computer-use-message")

    evidence.json("run-start.json", {
        "started_at": timing.started_at,
        "provider": provider,
        "model": model,
        "effort": effort,
        "permission_mode": DEFAULT_PERMISSION_MODE,
        "prompt_mode": DEFAULT_PROMPT_MODE,
        "source_commit": source_commit(workspace),
        "desktop_build": app_identity(app_path),
        "api_key_read": False,
        "api_key_output": False,
        "auth_token_output": False,
    })

    prompt = (
        "请只使用 ComputerUse 完成这个真实桌面闭环，并在 120 秒内结束：目标应用使用通用 target_id=workbuddy，"
        "WorkBuddy 当前未启动。第一步使用 action=observe 并携带 target_id=workbuddy；如果返回 launch_receipt 和绑定的目标窗口，"
        "不要再重复 launch_app 或 observe，直接检查这一次返回的最新目标截图。只有首次 observe 没有完成启动绑定时，才使用 "
        "action=launch_app、target_id=workbuddy，然后 observe 绑定的目标窗口。必须等待内容 ready，不能只因为窗口出现就操作；"
        "如果白屏，最多等待 12 秒并重新 observe，仍白屏就 stop。内容 ready 后点击目标应用的新建会话或新建任务（如果已经在新建页面则跳过），"
        "使用每个成功输入动作返回的最新 observation，不要无理由重复 observe；定位输入框，输入 1+1=2，直接检查 type 返回的最新截图，"
        "真实点击发送/提交按钮，等待一次回复；使用 wait 返回的最新截图确认真实回复，最后使用 ComputerUse stop。"
        "所有后续 click/type/key/send 必须使用绑定的 window_id；如果目标窗口变成 go-e2e、Chrome、Edge 或其他应用，"
        "立即 stop。只使用 ComputerUse，不要使用 Bash、脚本、osascript、System Events、screencapture、open 或其他工具；"
        "不要操作 go-e2e 自己；每个输入前必须使用最新 observe；任何失败或不确定立即 stop，不要重放或重复点击发送。"
    )

    if observe_only:
        prompt = ("只用 ComputerUse 做一次冷启动观察验收，不进行任何 click/type/key/hotkey 等输入。"
                  "第一步 action=observe,target_id=workbuddy，由 ComputerUse 启动绑定应用。"
                  "检查返回的真实目标截图是否 ready；白屏最多被动观察12秒，保存每张截图。"
                  "无论成功失败，最后 action=stop。不要预打开应用，不使用任何其他工具。")

    try:
        create_key = "computer-use-gpt6-sol-120s-" + uuid.uuid4().hex
        created = client.request("/tenant/session-control/sessions", "POST", {
            "session_key": create_key,
            "title": "Computer Use generic target 120s acceptance",
            "cwd": str(workspace),
            "provider": provider,
            "model": model,
            "permission_mode": DEFAULT_PERMISSION_MODE,
            "effort": effort,
            "prompt_mode": DEFAULT_PROMPT_MODE,
        }, timeout=min(8.0, max(1.0, deadline - clock())))
        session, status = get_status(created)
        ref = str(session.get("ref", ""))
        if not ref or ":" not in ref:
            raise AcceptanceError("launch", "session create response has no valid ref")
        source, key = ref.split(":", 1)
        session_path = f"/tenant/session-control/sessions/{source}/{key}"
        evidence.json("session-create.json", {"captured_at": iso_now(), "session": {
            "id": session.get("id"), "ref": ref, "provider": session.get("provider"),
            "model": session.get("model"), "effort": session.get("effort"), "status": status,
        }})
        evidence.json("session-binding.json", {"captured_at": iso_now(), "ref": ref, "path": session_path,
                                                    "session_id": session.get("id"), "provider": provider,
                                                    "model": model, "effort": effort})

        # The GET is the effective runtime gate. Do not trust only the create
        # request or the natural-language prompt to select a route.
        effective_session, _ = get_status(client.request(session_path, timeout=min(8.0, max(1.0, deadline - clock()))))
        effective = validate_effective_config(effective_session, provider, model, effort)
        effective.update({"requested_provider": provider, "requested_model": model, "requested_effort": effort,
                          "computer_use_image_route": "pending tool_result evidence", "session_ref": ref})
        evidence.json("effective-config.json", effective)

        evidence.json("preflight-workbuddy-state.json", {
            "source": "first ComputerUse observe; WorkBuddy must not be pre-opened by this wrapper",
            "target_id": TARGET_ID,
            "target_bundle_id": TARGET_BUNDLE_ID,
            "observed": False,
        })
        evidence.json("run-prompt.json", {"prompt_sha256": sha256_text(prompt), "prompt_length": len(prompt)})

        send_body = {"content": prompt, "provider": provider, "model": model,
                     "permission_mode": DEFAULT_PERMISSION_MODE, "effort": effort, "prompt_mode": DEFAULT_PROMPT_MODE}
        message_future = executor.submit(client.request, session_path + "/messages", "POST", send_body,
                                         min(130.0, max(1.0, deadline - clock() + 5.0)))
        evidence.json("message-dispatch.json", {"dispatched_at": iso_now(), "provider": provider, "model": model,
                                                  "effort": effort, "request_status": "asynchronous"})

        stop_sent = False
        stop_response: Any = None
        while clock() < deadline:
            remaining = deadline - clock()
            try:
                snapshot, status = get_status(client.request(session_path, timeout=min(5.0, max(1.0, remaining))))
            except Exception as exc:  # Keep polling until the deadline; do not guess terminal state.
                status_history.append({"at": iso_now(), "elapsed_seconds": round(clock() - start, 3),
                                       "status": "status_error", "error": str(exc)[:200]})
                if clock() >= operational and not stop_sent:
                    break
                sleep(min(poll_interval, max(0.05, operational - clock())))
                continue
            status_history.append({"at": iso_now(), "elapsed_seconds": round(clock() - start, 3), "status": status})
            if status in TERMINAL_STATUSES and (clock() - start) > 0.2:
                final_status = status
                break
            if clock() >= operational and not stop_sent:
                stop_started = clock()
                try:
                    stop_response = client.request(session_path + "/stop", "POST", {}, timeout=min(5.0, max(1.0, deadline - clock())))
                except Exception as exc:
                    stop_response = {"error": str(exc)[:200]}
                timing.stop_duration = max(0.0, clock() - stop_started)
                evidence.json("timeout-stop-response.json", {"captured_at": iso_now(), "response": unwrap(stop_response)})
                stop_sent = True
            sleep(min(poll_interval, max(0.05, deadline - clock())))

        if not final_status or final_status == "failed":
            snapshot, final_status = get_status(client.request(session_path, timeout=min(5.0, max(1.0, deadline - clock()))))
        if final_status not in TERMINAL_STATUSES:
            raise AcceptanceError("stop", f"session did not reach a terminal status before the 120-second deadline: {final_status!r}")

        evidence.json("status-history.json", {"captured_at": iso_now(), "history": status_history})
        try:
            if message_future is not None and message_future.done():
                message_response = message_future.result()
                evidence.json("message-response.json", {"captured_at": iso_now(), "response": unwrap(message_response)})
            else:
                evidence.json("message-response.json", {"captured_at": iso_now(), "request_status": "not_completed_before_deadline"})
        except Exception as exc:
            evidence.json("message-response.json", {"captured_at": iso_now(), "request_status": "error", "error": str(exc)[:200]})

        remaining = deadline - clock()
        if remaining <= 0.25:
            raise AcceptanceError("stop", "the evidence deadline expired before conversation trace could be fetched")
        events = latest_conversation_events(client, session_path, timeout=min(8.0, remaining - 0.1))
        actions = tool_events(events)
        log_metrics = execution_log_metrics(events, actions)
        # Persist the complete redacted execution trace before downloading any
        # screenshot. A bad/expired image must never erase the action history.
        evidence.json("action-receipts.json", {"schema_version": "computer-use-action-receipts.v1", "actions": actions})
        evidence.json("conversation-trace.json", safe_event_trace(events, actions))
        evidence.json("execution-log-metrics.json", log_metrics)
        validate_run_actions(actions, timing.started_at, iso_now())
        output_paths, invalid_assets = save_observation_assets(client, evidence, actions, deadline)
        # Update preflight evidence from the first actual observation without
        # reading or OCRing pixels; the screenshot itself is the authority.
        first_observation = next((item for item in actions if item.get("action") == "observe" and item.get("computer_observation")), None)
        if first_observation:
            evidence.json("preflight-workbuddy-state.json", {
                "source": "first ComputerUse observe",
                "target_id": TARGET_ID,
                "target_bundle_id": TARGET_BUNDLE_ID,
                "observation_id": first_observation["computer_observation"].get("observation_id"),
                "workbuddy_bundle_seen": first_observation.get("bundle_id") == TARGET_BUNDLE_ID,
                "screenshot_path": output_paths.get(first_observation["computer_observation"].get("observation_id", ""), ""),
            })
        launch_item = next((item for item in actions if item.get("action") == "launch_app"), None)
        if launch_item:
            timing.launch_started = parse_iso(str(launch_item.get("call_at", "")))
            timing.launch_finished = parse_iso(str(launch_item.get("result_at", "")))
            first_target_observe = next(
                (item for item in actions if item.get("action") == "observe" and item.get("call_at", "") > launch_item.get("result_at", "")),
                None,
            )
            if first_target_observe:
                timing.binding_started = timing.launch_finished
                timing.binding_finished = parse_iso(str(first_target_observe.get("result_at", "")))
                timing.readiness_finished = timing.binding_finished
        timing.click_duration = sum(float(item.get("duration_seconds", 0.0)) for item in actions if item.get("action") == "click")
        timing.type_duration = sum(float(item.get("duration_seconds", 0.0)) for item in actions if item.get("action") == "type")
        click_durations = [float(item.get("duration_seconds", 0.0)) for item in actions if item.get("action") == "click"]
        timing.send_duration = click_durations[-1] if click_durations else 0.0
        timing.reply_wait_duration = sum(float(item.get("duration_seconds", 0.0)) for item in actions if item.get("action") == "wait")
        timing.stop_duration = sum(float(item.get("duration_seconds", 0.0)) for item in actions if item.get("action") == "stop")

        # Persist evidence before semantic validation. A failed run must still
        # leave the exact redacted action trail and screenshots that explain its
        # launch/window/content/reply stage.
        image_route = any(item.get("computer_observation", {}).get("media_type") == "image/png" for item in actions)
        effective["computer_use_image_route"] = "image/png" if image_route else "missing"
        evidence.json("effective-config.json", effective)
        reply_candidates = [
            item for item in actions
            if item.get("action") == "wait"
            and isinstance(item.get("computer_observation"), dict)
            and item["computer_observation"].get("observation_id") in output_paths
        ]
        if reply_candidates:
            reply_id = reply_candidates[-1]["computer_observation"]["observation_id"]
            shutil.copyfile(output_paths[reply_id], evidence.root / "reply-screenshot.png")

        validation_error: Optional[AcceptanceError] = None
        try:
            binding = (validate_observe_only(actions, output_paths, final_status) if observe_only
                       else validate_actions(actions, output_paths, final_status, invalid_assets=invalid_assets))
        except AcceptanceError as exc:
            validation_error = exc
            binding = {
                "target_id": TARGET_ID,
                "action_count": len(actions),
                "actions": [item.get("action", "") for item in actions],
                "screenshot_count": len(output_paths),
                "validation_error": {"stage": exc.stage, "message": str(exc)},
            }
        evidence.json("sanitized-timeline.json", {
            "schema_version": "computer-use-sanitized-timeline.v1", "provider": provider, "model": model,
            "effort": effort, "session_ref": ref, "session_status": final_status, "binding": binding,
            "actions": actions,
            "invalid_observation_assets": invalid_assets,
            "execution_log_metrics": log_metrics,
        })
        if validation_error is not None:
            raise validation_error
        if not image_route:
            raise AcceptanceError("content readiness", "ComputerUse did not produce an image-capable observation result")
        end = clock()
        timing_data = build_timing(timing, final_status, end, provider, model, effort)
        timing_data["execution_log_metrics"] = log_metrics
        evidence.json("timing.json", timing_data)
        return {"status": "passed", "final_status": final_status, "timing": timing_data, "binding": binding}
    except AcceptanceError as exc:
        error_stage = exc.stage
        final_status = final_status if final_status in TERMINAL_STATUSES else "failed"
        evidence.json("failure.json", {"status": "failed", "stage": exc.stage, "message": str(exc), "at": iso_now()})
        evidence.json("status-history.json", {"captured_at": iso_now(), "history": status_history})
        timing_data = build_timing(timing, final_status, clock(), provider, model, effort, error_stage)
        timing_data["execution_log_metrics"] = log_metrics
        evidence.json("timing.json", timing_data)
        return {"status": "failed", "final_status": final_status, "failure_stage": exc.stage, "message": str(exc), "timing": timing_data}
    except Exception as exc:
        error_stage = "launch" if not session_path else "conversation_trace"
        evidence.json("failure.json", {"status": "failed", "stage": error_stage, "message": str(exc)[:500], "at": iso_now()})
        evidence.json("status-history.json", {"captured_at": iso_now(), "history": status_history})
        timing_data = build_timing(timing, final_status, clock(), provider, model, effort, error_stage)
        timing_data["execution_log_metrics"] = log_metrics
        evidence.json("timing.json", timing_data)
        return {"status": "failed", "final_status": final_status, "failure_stage": error_stage, "message": str(exc)[:500], "timing": timing_data}
    finally:
        executor.shutdown(wait=False, cancel_futures=True)


def parse_args(argv: Optional[list[str]] = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT.with_name(DEFAULT_OUTPUT.name + "-" + dt.datetime.now().strftime("%H%M%S") + "-" + uuid.uuid4().hex[:8]))
    parser.add_argument("--observe-only", action="store_true", help="validate launch/binding/capture/stop without any input")
    parser.add_argument("--workspace", type=Path, default=DEFAULT_WORKSPACE)
    parser.add_argument("--app", type=Path, default=DEFAULT_APP_PATH)
    parser.add_argument("--port", type=int, default=0, help="use an explicit local server port instead of process discovery")
    parser.add_argument("--auth-token", default="", help=argparse.SUPPRESS)
    parser.add_argument("--provider", default=DEFAULT_PROVIDER)
    parser.add_argument("--model", default=DEFAULT_MODEL)
    parser.add_argument("--effort", default=DEFAULT_EFFORT, choices=("low", "medium", "high", "off"))
    parser.add_argument("--total-budget", type=float, default=DEFAULT_TOTAL_BUDGET_SECONDS)
    parser.add_argument("--operational-deadline", type=float, default=DEFAULT_OPERATIONAL_DEADLINE_SECONDS)
    parser.add_argument("--poll-interval", type=float, default=DEFAULT_POLL_INTERVAL_SECONDS)
    return parser.parse_args(argv)


def main(argv: Optional[list[str]] = None) -> int:
    args = parse_args(argv)
    try:
        validate_build_identity(args.workspace, args.app)
        subprocess.run(["codesign", "--verify", "--deep", "--strict", str(args.app)], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if args.port and args.auth_token:
            server = LocalServer(0, 0, args.port, args.auth_token)
        else:
            server = find_local_server(args.workspace)
        result = run_acceptance(
            client=APIClient(server.port, server.token),
            evidence=Evidence(args.output),
            workspace=args.workspace,
            app_path=args.app,
            provider=args.provider,
            model=args.model,
            effort=args.effort,
            total_budget=args.total_budget,
            operational_deadline=min(args.operational_deadline, args.total_budget),
            poll_interval=args.poll_interval,
            observe_only=args.observe_only,
        )
        print(json.dumps({"status": result.get("status"), "final_status": result.get("final_status"),
                          "failure_stage": result.get("failure_stage", ""), "evidence": str(args.output),
                          "total_elapsed_seconds": result.get("timing", {}).get("total_elapsed_seconds")}, ensure_ascii=False))
        return 0 if result.get("status") == "passed" else 1
    except Exception as exc:
        print(json.dumps({"status": "failed", "failure_stage": "launch", "message": str(exc)[:500],
                          "evidence": str(args.output)}, ensure_ascii=False), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
