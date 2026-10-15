"""Repeatable destructive checks, restricted to newly created acceptance stands."""

import concurrent.futures
import json
import os
import subprocess
import sys
import time
import threading
import uuid
from pathlib import Path

from scripts.testing.controller import ARTIFACTS, ROOT, RUNTIME, Stand, save_json


class Acceptance:
    def __init__(self):
        self.id = "check-" + uuid.uuid4().hex[:8]
        self.output = ARTIFACTS / self.id
        self.output.mkdir(parents=True)
        self.records = []
        self.record_lock = threading.Lock()
        # Record ownership before creation; teardown may delete only this stand.
        self.stand = Stand(self.id, self.id + "-bootstrap")
        if not self.stand.created:
            raise RuntimeError("acceptance ID already owns containers; refusing reuse")

    def start(self, mode, suffix, stand_id=None, fault=None, gate=None):
        run_id = self.id + "-" + suffix
        env = dict(os.environ, RUN_ID=run_id, STAND_ID=stand_id or self.id,
                   TEST_FAULT=fault or "", TEST_FAULT_GATE=gate or "")
        log_suffix = suffix + (".restore" if mode == "restore" else "")
        output = (self.output / (log_suffix + ".log")).open("wb")
        process = subprocess.Popen([sys.executable, "-m", "scripts.testing.controller", mode],
                                   cwd=ROOT, env=env, stdout=output, stderr=subprocess.STDOUT)
        return process, run_id, output

    def finish(self, task, expected=0):
        process, run_id, output = task
        try:
            code = process.wait(timeout=300)
        finally:
            output.close()
        if (code == 0) != (expected == 0):
            raise AssertionError(f"{run_id}: exit {code}, expected {'success' if expected == 0 else 'failure'}")
        record = {"run": run_id, "exit": code}
        recovery = ARTIFACTS / run_id / "recovery.json"
        if recovery.exists():
            restored = json.loads(recovery.read_text())["restored"]
            if not restored:
                raise AssertionError(f"{run_id}: snapshot not restored")
            record["restored"] = True
        with self.record_lock:
            self.records.append(record)
            save_json(self.output / "acceptance.json", self.records)
        return run_id

    def execute(self, mode, suffix, **kwargs):
        expected = kwargs.pop("expected", 0)
        return self.finish(self.start(mode, suffix, **kwargs), expected)

    def interrupt_stage(self, stage):
        task = self.start("all", "killed-" + stage, gate=stage)
        process, run_id, _ = task
        gate = RUNTIME / "runs" / run_id / (stage + "-written")
        deadline = time.monotonic() + 180
        while not gate.exists():
            if process.poll() is not None or time.monotonic() >= deadline:
                raise AssertionError(f"{run_id}: never reached committed-data gate")
            time.sleep(0.2)
        name = f"marketplace-test-{self.id}-{run_id}-{stage}"
        subprocess.run(["docker", "kill", "--signal=KILL", name], check=True, stdout=subprocess.DEVNULL)
        self.finish(task, expected=1)
        statuses = json.loads((ARTIFACTS / run_id / "stages.json").read_text())
        if statuses[stage] != "broken" or (stage == "integration" and statuses["e2e"] != "skipped"):
            raise AssertionError(f"interruption misreported: {statuses}")

    def interrupt_controller(self, stage="integration"):
        shared = stage == "e2e"
        suffix = "killed-shared-controller" if shared else "killed-controller"
        gate_name = "shared" if shared else "integration"
        manual_email = "manual-" + self.id + "@example.test"
        if shared:
            before_api = self.stand.compose("ps", "-q", "api", capture_output=True, text=True).stdout.strip()
            body = json.dumps({"email": manual_email, "password": "Synthetic-password-42",
                               "full_name": "Manual sentinel", "role": "buyer"})
            self.stand.compose("run", "--rm", "--no-deps", "--entrypoint", "curl", "runner",
                               "--fail", "--silent", "-H", "Content-Type: application/json", "-d", body,
                               "http://api:8080/api/v1/auth/register", capture_output=True)
        task = self.start(stage, suffix, gate=gate_name)
        process, run_id, output = task
        deadline = time.monotonic() + 180
        gate = RUNTIME / "runs" / run_id / (gate_name + "-written")
        while not gate.exists():
            if process.poll() is not None or time.monotonic() >= deadline:
                raise AssertionError("controller-death check did not reach committed fixture")
            time.sleep(0.2)
        process.kill()
        process.wait(timeout=10)
        output.close()
        self.execute("integration", "blocked-" + suffix, expected=1)
        self.execute("restore", suffix)
        statuses = json.loads((ARTIFACTS / run_id / "stages.json").read_text())
        if statuses.get(stage) != "broken":
            raise AssertionError(f"recovery did not finalize interrupted stage: {statuses}")
        name = f"marketplace-test-{self.id}-{run_id}-{stage}"
        result = subprocess.run(["docker", "inspect", name], capture_output=True)
        if result.returncode == 0:
            raise AssertionError("orphan runner survived recovery")
        if shared:
            after_api = self.stand.compose("ps", "-q", "api", capture_output=True, text=True).stdout.strip()
            manual_count = self.stand.compose("exec", "-T", "db", "psql", "-U", "postgres", "-d", "marketplace",
                "-At", "--set", "email=" + manual_email, input="SELECT count(*) FROM users WHERE email=:'email';",
                capture_output=True, text=True).stdout.strip()
            if before_api != after_api or manual_count != "1":
                raise AssertionError("shared recovery changed the API or deleted manual data")
            manifests = list((RUNTIME / "runs" / run_id / "fixtures").glob("*.json"))
            if not manifests or any(not json.loads(path.read_text())["cleaned"] for path in manifests):
                raise AssertionError("shared recovery left fixture data")

    def run(self):
        try:
            self.execute("stand-up", "bootstrap")
            baseline = None
            counts = None
            for index in range(1, 4):
                run_id = self.execute("integration", "repeat-" + str(index))
                record = json.loads((ARTIFACTS / run_id / "recovery.json").read_text())
                if baseline is not None and baseline != record["fingerprint"]:
                    raise AssertionError("baseline drift across repeated integration runs")
                baseline = record["fingerprint"]
                from scripts.testing.results import junit_summary
                current = junit_summary(ARTIFACTS / run_id / "raw/integration.xml")
                if counts is not None and current != counts:
                    raise AssertionError("integration outcome changed across repeats")
                counts = current
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as executor:
                tasks = [executor.submit(self.execute, "all", "parallel-" + side, stand_id=self.id + "-" + side)
                         for side in ["a", "b"]]
                for task in tasks:
                    task.result()
            failed = self.execute("all", "failed-unit", fault="unit", expected=1)
            statuses = json.loads((ARTIFACTS / failed / "stages.json").read_text())
            if statuses != {"unit": "failed", "integration": "skipped", "e2e": "skipped"}:
                raise AssertionError(f"fail-fast: {statuses}")
            self.execute("e2e", "partial-fixture", fault="fixture", expected=1)
            self.execute("guard-cleanup", "foreign-references")
            for stage in ["integration", "e2e"]:
                self.interrupt_stage(stage)
            self.interrupt_controller()
            self.interrupt_controller("e2e")
            successful = self.execute("all", "after-interruption")
            trend = json.loads((ARTIFACTS / successful / "allure-report/history/history-trend.json").read_text())
            if len(trend) < 2:
                raise AssertionError("Allure history missing previous run")
            self.execute("traffic", "traffic")
            print(f"Acceptance passed: {self.output}")
        finally:
            self.stand.teardown()


if __name__ == "__main__":
    Acceptance().run()
