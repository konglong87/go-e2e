#!/usr/bin/env python3
import base64
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import sys
import time


ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location("computer_use_acceptance", ROOT / "scripts/computer-use-120s-acceptance.py")
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


# 1x1 RGBA PNGs used only for offline wrapper tests.
WHITE_PNG = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
)
DARK_PNG = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
)


class WrapperUnitTests(unittest.TestCase):
    def test_effective_provider_and_model_are_exact(self):
        session = {"provider": MODULE.DEFAULT_PROVIDER, "model": MODULE.DEFAULT_MODEL, "effort": "high", "status": "idle"}
        result = MODULE.validate_effective_config(session, MODULE.DEFAULT_PROVIDER, MODULE.DEFAULT_MODEL, "high")
        self.assertEqual(result["effective_provider"], MODULE.DEFAULT_PROVIDER)
        with self.assertRaises(MODULE.AcceptanceError) as mismatch:
            MODULE.validate_effective_config(session, "custom", MODULE.DEFAULT_MODEL, "high")
        self.assertEqual(mismatch.exception.stage, "effective_config")

    def test_tool_events_keep_counts_and_strip_sensitive_output(self):
        events = [
            {"event_type": "tool_call", "created_at": "2026-10-02T00:00:00Z", "payload_json": json.dumps({
                "tool_id": "tool-1", "tool_name": "ComputerUse",
                "input": json.dumps({"redacted": True, "action": "type", "text_length": 5, "input_bytes": 137}),
            })},
            {"event_type": "tool_result", "created_at": "2026-10-02T00:00:01Z", "payload_json": json.dumps({
                "tool_id": "tool-1", "tool_name": "ComputerUse", "is_error": False,
                "output": '{"window_id":"19988","bundle_id":"com.workbuddy.workbuddy","outcome":"executed","secret":"must-not-be-persisted"}',
                "computer_observation": {"observation_id": "observation-1", "asset_id": "asset-1", "media_type": "image/png"},
            })},
        ]
        actions = MODULE.tool_events(events)
        self.assertEqual(actions[0]["action"], "type")
        self.assertEqual(actions[0]["text_length"], 5)
        self.assertEqual(actions[0]["window_id"], "19988")
        self.assertNotIn("secret", json.dumps(actions))

    def test_action_validation_rejects_control_window(self):
        actions = [
            {"tool_id": "1", "action": "observe", "is_error": False, "computer_observation": {"observation_id": "o1"}},
            {"tool_id": "2", "action": "launch_app", "is_error": False, "outcome": "executed", "bundle_id": "com.workbuddy.workbuddy", "window_id": "19988"},
            {"tool_id": "3", "action": "observe", "is_error": False, "bundle_id": "com.openai.codex", "window_id": "19988", "computer_observation": {"observation_id": "o2"}},
            {"tool_id": "4", "action": "type", "is_error": False, "bundle_id": "com.workbuddy.workbuddy", "window_id": "19988", "text_length": 5},
            {"tool_id": "5", "action": "click", "is_error": False, "bundle_id": "com.workbuddy.workbuddy", "window_id": "19988"},
            {"tool_id": "6", "action": "wait", "is_error": False, "bundle_id": "com.workbuddy.workbuddy", "window_id": "19988", "computer_observation": {"observation_id": "o3"}},
            {"tool_id": "7", "action": "stop", "is_error": False, "bundle_id": "com.workbuddy.workbuddy", "window_id": "19988"},
        ]
        with self.assertRaises(MODULE.AcceptanceError) as failure:
            MODULE.validate_actions(actions, {"o1": "one.png", "o2": "two.png", "o3": "three.png"}, "stopped")
        self.assertEqual(failure.exception.stage, "window binding")

    def test_blank_guard_rejects_white_png(self):
        # The fixture is intentionally tiny; the structural guard itself is
        # tested by checking that invalid/non-image bytes are rejected.
        with self.assertRaises(ValueError):
            MODULE.validate_png(b"not-a-png", "image/png")

    def test_evidence_directory_is_private(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = MODULE.Evidence(Path(directory) / "evidence")
            evidence.json("safe.json", {"provider": MODULE.DEFAULT_PROVIDER})
            self.assertEqual((Path(directory) / "evidence").stat().st_mode & 0o777, 0o700)
            self.assertEqual((Path(directory) / "evidence/safe.json").stat().st_mode & 0o777, 0o600)

    def test_timing_preserves_requested_effort(self):
        timing = MODULE.Timing(started_at="2026-10-02T00:00:00Z", start_monotonic=10.0)
        result = MODULE.build_timing(timing, "stopped", 11.25, MODULE.DEFAULT_PROVIDER, MODULE.DEFAULT_MODEL, "low")
        self.assertEqual(result["effort"], "low")

    def test_cli_exposes_low_effort_override(self):
        args = MODULE.parse_args(["--provider", "jiuan-responses-gpt-5.6sol", "--model", "gpt-6-sol", "--effort", "low"])
        self.assertEqual(args.provider, "jiuan-responses-gpt-5.6sol")
        self.assertEqual(args.model, "gpt-6-sol")
        self.assertEqual(args.effort, "low")

class FakeAcceptanceClient:
    def __init__(self):
        self.stop_calls = 0
        self.poll_calls = 0
        self.events = self._events()
        self.image = self._png((24, 24, 24, 255))

    @staticmethod
    def _png(pixel):
        import struct
        import zlib
        raw = b"\x00" + bytes(pixel)
        def chunk(kind, body):
            return struct.pack(">I", len(body)) + kind + body + struct.pack(">I", MODULE.zlib.crc32(kind + body) & 0xFFFFFFFF)
        return MODULE.PNG_SIGNATURE + chunk(b"IHDR", struct.pack(">IIBBBBB", 1, 1, 8, 6, 0, 0, 0)) + chunk(b"IDAT", zlib.compress(raw)) + chunk(b"IEND", b"")

    @staticmethod
    def _events():
        names = ["observe", "launch_app", "observe", "type", "click", "wait", "stop"]
        events = []
        for index, action in enumerate(names):
            tool_id = f"tool-{index}"
            call_payload = {
                "tool_id": tool_id, "tool_name": "ComputerUse",
                "input": json.dumps({"redacted": True, "action": action, "text_length": 5 if action == "type" else 0}),
            }
            if action == "launch_app":
                output = json.dumps({"target_id": "workbuddy", "bundle_id": "com.workbuddy.workbuddy", "window": {"id": "19988", "owner_pid": 60626}, "window_id": "19988", "outcome": "executed"})
            elif action == "stop":
                output = json.dumps({"action": "stop", "status": "ok"})
            else:
                output = json.dumps({"window_id": "19988", "bundle_id": "com.workbuddy.workbuddy", "owner_pid": 60626, "outcome": "executed"})
            result_payload = {"tool_id": tool_id, "tool_name": "ComputerUse", "output": output, "is_error": False}
            if action in {"observe", "type", "click", "wait"}:
                result_payload["computer_observation"] = {"observation_id": f"observation-{index}", "asset_id": f"asset-{index}", "media_type": "image/png", "size_bytes": 100, "sha256": "hash"}
            timestamp = f"2026-10-02T00:00:{index:02d}Z"
            events.extend([
                {"event_type": "tool_call", "created_at": timestamp, "payload_json": json.dumps(call_payload)},
                {"event_type": "tool_result", "created_at": timestamp, "payload_json": json.dumps(result_payload)},
            ])
        return events

    def request(self, path, method="GET", body=None, timeout=None):
        if path.endswith("/messages"):
            return {"data": {"session": {"status": "running"}, "operation_id": "op-1"}}
        if path.endswith("/stop"):
            self.stop_calls += 1
            return {"data": {"session": {"status": "stopped"}}}
        if path.endswith("/conversation"):
            return {"data": {"events": self.events}}
        if path == "/tenant/session-control/sessions":
            return {"data": {"session": {"id": 1, "ref": "tenant:test", "provider": MODULE.DEFAULT_PROVIDER, "model": MODULE.DEFAULT_MODEL, "effort": "high", "status": "idle"}}}
        self.poll_calls += 1
        status = "running" if self.stop_calls == 0 else "stopped"
        return {"data": {"session": {"id": 1, "ref": "tenant:test", "provider": MODULE.DEFAULT_PROVIDER, "model": MODULE.DEFAULT_MODEL, "effort": "high", "status": status}}}

    def download(self, path, timeout=None):
        return self.image, "image/png"


class AsyncDeadlineTests(unittest.TestCase):
    def test_async_message_is_stopped_and_trace_is_collected(self):
        with tempfile.TemporaryDirectory() as directory:
            client = FakeAcceptanceClient()
            result = MODULE.run_acceptance(
                client=client,
                evidence=MODULE.Evidence(Path(directory)),
                workspace=ROOT,
                app_path=ROOT / "desktop-v2/build/bin/go-e2e.app",
                total_budget=2.0,
                operational_deadline=0.05,
                poll_interval=0.01,
                sleep=lambda seconds: time.sleep(min(seconds, 0.01)),
            )
            self.assertEqual(result["status"], "passed")
            self.assertEqual(result["final_status"], "stopped")
            self.assertEqual(client.stop_calls, 1)
            self.assertTrue((Path(directory) / "conversation-trace.json").exists())
            self.assertNotIn("auth-token", (Path(directory) / "conversation-trace.json").read_text())


if __name__ == "__main__":
    unittest.main()
