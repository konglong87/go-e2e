#!/usr/bin/env python3
"""Opt-in safety acceptance against the actual running go-e2e host/helper.

Wait interruptions do not exercise a held mouse/key. Permission revocation is
not performed here: it requires a separate user-controlled OS authorization run.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import importlib.util
import json
import os
from pathlib import Path
import signal
import subprocess
import time
import uuid

spec = importlib.util.spec_from_file_location("fixture_runner", Path(__file__).with_name("computer-acceptance-fixture.py"))
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)

class SafetyRun:
    def __init__(self, fixture_file):
        self.d = mod.module.Driver()
        self.fixture = mod.FixtureRun(fixture_file, self.d)
        self.results = []

    def record(self, name, passed, **data):
        self.results.append(dict(case=name, passed=passed, **data))
        (self.d.output / "safety-results.json").write_text(json.dumps(self.results, ensure_ascii=False, indent=2))
        print("PASS" if passed else "FAIL", name, flush=True)

    def restart(self):
        if self.d.session:
            self.d.call("stop", session_id=self.d.session)
        self.d.start()

    def action(self, obs, **fields):
        return dict(id=str(uuid.uuid4()), session_id=self.d.session, observation_id=obs["id"], **fields)

    def denied(self, action):
        baseline = self.fixture.snapshot()["state"]["inputValue"]
        r = self.d.call("execute", session_id=self.d.session, action=action)
        time.sleep(.3)
        unchanged = self.fixture.snapshot()["state"]["inputValue"] == baseline
        return r, unchanged and bool(r.get("error")) and r.get("data", {}).get("outcome") == "rejected"

    def expired(self):
        self.restart()
        obs = self.d.observe("safety-expired-before")
        time.sleep(11)
        r, passed = self.denied(self.action(obs, kind="type", text="MUST_NOT_APPEAR"))
        self.record("expired-observation", passed, response=r)

    def stale(self):
        self.restart()
        old = self.d.observe()
        self.d.observe()
        r, passed = self.denied(self.action(old, kind="type", text="MUST_NOT_APPEAR"))
        self.record("superseded-observation", passed, response=r)

    def interrupted(self, control):
        self.restart()
        obs = self.d.observe()
        action = self.action(obs, kind="wait", duration_ms=5000)
        with ThreadPoolExecutor(max_workers=1) as pool:
            future = pool.submit(self.d.call, "execute", session_id=self.d.session, action=action)
            time.sleep(.25)
            began = time.monotonic()
            response = self.d.call(control, session_id=self.d.session)
            inflight = future.result(timeout=3)
            elapsed = time.monotonic() - began
        snapshot = self.d.control("snapshot")
        next_response, denied = self.denied(self.action(obs, kind="type", text="MUST_NOT_APPEAR"))
        wanted = "paused" if control == "pause" else "stopped"
        safe = snapshot["state"] == wanted and elapsed < 2 and denied and inflight.get("data", {}).get("outcome") != "executed"
        self.record(control + "-during-wait", safe, elapsed_seconds=elapsed, control_response=response,
                    action_response=inflight, next_response=next_response, state=snapshot["state"])
        if control == "pause":
            resume = self.d.call("resume", session_id=self.d.session)
            observed = self.d.call("observe", session_id=self.d.session) if not resume.get("error") else {}
            observed.pop("data", None) # no PNGs in this summary
            self.record("resume-after-interrupted-wait", not resume.get("error") and not observed.get("error"), response=resume, observe_error=observed.get("error"))

    def crash(self):
        self.restart()
        obs = self.d.observe()
        app_path = str(Path(__file__).resolve().parents[1] / "desktop-v2/build/bin/go-e2e.app/Contents")
        expected = app_path + "/Helpers/ComputerHelper.app/Contents/MacOS/computer-helper-macos"
        main = app_path + "/MacOS/go-e2e-desktop"
        rows = [row.strip().split(None, 2) for row in subprocess.check_output(["ps", "-axo", "pid,ppid,comm"], text=True).splitlines()[1:]]
        hosts = {int(row[0]) for row in rows if len(row) == 3 and row[2] == main}
        helpers = [int(row[0]) for row in rows if len(row) == 3 and row[2] == expected and int(row[1]) in hosts]
        if len(helpers) != 1:
            raise RuntimeError("refusing crash injection: expected exactly one child of this desktop host")
        with ThreadPoolExecutor(max_workers=1) as pool:
            future = pool.submit(self.d.call, "execute", session_id=self.d.session, action=self.action(obs, kind="wait", duration_ms=5000))
            time.sleep(.25)
            os.kill(helpers[0], signal.SIGKILL)
            response = future.result(timeout=3)
        later = self.d.call("observe", session_id=self.d.session)
        later.pop("data", None)
        safe = bool(response.get("error")) and response.get("data", {}).get("outcome") == "unknown" and bool(later.get("error"))
        self.record("native-helper-crash", safe, response=response, later_response=later)
        self.restart()
        recovery = self.d.observe("safety-crash-recovered")
        self.record("new-session-after-helper-crash", bool(recovery.get("id")))

    def run(self, crash):
        try:
            self.expired(); self.stale(); self.interrupted("pause"); self.interrupted("stop")
            if crash:
                self.crash()
        finally:
            self.d.call("stop", session_id=self.d.session)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--connection", type=Path, default=Path.home() / ".go-e2e/computer-acceptance/fixture.json")
    parser.add_argument("--crash-helper", action="store_true", help="explicitly kill only this host's bundled helper during wait")
    args = parser.parse_args()
    runner = SafetyRun(args.connection)
    runner.run(args.crash_helper)
    if any(not case["passed"] for case in runner.results):
        raise SystemExit("safety acceptance has failing cases; inspect safety-results.json")

if __name__ == "__main__":
    main()
