"""Host-side owner of Docker processes, snapshots and recovery (stdlib only)."""

import argparse
import contextlib
import fcntl
import hashlib
import json
import os
import re
import shutil
import signal
import subprocess
import tarfile
import time
import uuid
from pathlib import Path

from scripts.testing.results import stage_result

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / "artifacts/lab2"
RUNTIME = Path(os.getenv("TEST_RUNTIME_ROOT", "/tmp/marketplace-test-runtime"))


def runner_image():
    digest = hashlib.sha256()
    for name in ["tests/Dockerfile", "tests/requirements.txt", "tests/runner.sh"]:
        digest.update(name.encode() + b"\0" + (ROOT / name).read_bytes())
    return "marketplace-test-runner:" + digest.hexdigest()[:20]


def resource_id(value):
    if not re.fullmatch(r"[a-z0-9][a-z0-9_-]{0,80}", value):
        raise ValueError(f"unsafe resource ID: {value!r}")
    return value


def dump_fingerprint(data):
    lines = [line for line in data.splitlines() if line.strip() and not line.startswith((b"--", b"\\restrict", b"\\unrestrict"))]
    return hashlib.sha256(b"\n".join(sorted(lines))).hexdigest()


def permissions_fingerprint(schema):
    # CHECK-expression deparse can change after restore without changing semantics.
    permissions = [line for line in schema.splitlines()
                   if line.startswith((b"GRANT ", b"REVOKE ")) or
                   (line.startswith(b"ALTER ") and b" OWNER TO " in line)]
    return dump_fingerprint(b"\n".join(permissions))


def save_json(path, value):
    temporary = path.with_suffix(".tmp")
    temporary.write_text(json.dumps(value, indent=2))
    temporary.replace(path)


def check_ownership(record, stand_id, project, existing):
    if (existing and record is None) or (record is not None and
            (record.get("stand") != stand_id or record.get("project") != project)):
        raise ValueError("Compose resources lack a matching test-stand ownership record; refusing mutation")


def require_recovered(stand_id, restoring=None):
    # Runtime is authoritative when the controller died before its final publish.
    records = {path.parent.name: path for path in ARTIFACTS.glob("*/recovery.json")}
    records.update({path.parent.name: path for path in (RUNTIME / "runs").glob("*/recovery.json")})
    for run_id, path in records.items():
        record = json.loads(path.read_text())
        if (record.get("stand") == stand_id and (record.get("snapshot_ready") or record.get("mode") == "shared") and
                not record.get("restored") and run_id != restoring):
            raise RuntimeError(f"stand {stand_id} needs recovery: make test-restore RUN_ID={run_id}")


def pipeline(directory, stages, execute):
    statuses = {}
    stopped = False
    for stage in stages:
        if stopped:
            statuses[stage] = "skipped"
            stage_result(directory / "allure-results", stage, "skipped", "previous stage did not pass")
        else:
            statuses[stage] = "running"
            save_json(directory / "stages.json", statuses)
            try:
                execute(stage)
                statuses[stage] = "passed"
                stage_result(directory / "allure-results", stage, "passed")
            except BaseException as error:
                # Infrastructure markers do not replace the real product-test results.
                statuses[stage] = "failed" if isinstance(error, subprocess.CalledProcessError) and error.returncode < 128 else "broken"
                stage_result(directory / "allure-results", stage, statuses[stage], str(error))
                stopped = True
        save_json(directory / "stages.json", statuses)
    return statuses


class Stand:
    def __init__(self, stand_id, run_id, allow_existing=False):
        self.id = resource_id(stand_id)
        self.run_id = resource_id(run_id)
        self.project = "marketplace-test-" + self.id
        self.directory = RUNTIME / "stands" / self.id
        self.run = RUNTIME / "runs" / self.run_id
        self.artifacts = ARTIFACTS / self.run_id
        self.directory.mkdir(parents=True, exist_ok=True)
        if not allow_existing and ((self.run / "environment.json").exists() or self.artifacts.exists()):
            raise ValueError("RUN_ID already exists; choose a new ID (restore is the only reuse command)")
        self.run.mkdir(parents=True, exist_ok=True)
        (self.run / "workspace").mkdir(exist_ok=True)
        (self.run / "allure-results").mkdir(exist_ok=True)
        cache = RUNTIME / "cache"
        (cache / "mod").mkdir(parents=True, exist_ok=True)
        (cache / "build").mkdir(parents=True, exist_ok=True)
        self.env = dict(os.environ, STAND_PATH=str(self.directory), RUN_PATH=str(self.run), RUN_ID=self.run_id, CACHE_PATH=str(cache))
        self.env.setdefault("BUILDX_CONFIG", "/tmp/marketplace-buildx")
        self.env["TEST_COMMIT_SHA"] = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
        self.metadata = self.directory / "stand.json"
        self.active = None
        self.refresh_ownership()

    def refresh_ownership(self):
        record = json.loads(self.metadata.read_text()) if self.metadata.exists() else None
        if record and record.get("port"):
            self.env["UI_PORT"] = str(record["port"])
        selector = "label=com.docker.compose.project=" + self.project
        existing = any(self.docker(*command, "--filter", selector, "--format", format,
                                  capture_output=True, text=True).stdout.strip()
                       for command, format in [(["ps", "-a"], "{{.ID}}"), (["network", "ls"], "{{.ID}}"),
                                               (["volume", "ls"], "{{.Name}}")])
        self.created = not existing
        check_ownership(record, self.id, self.project, existing)
        self.env["RUNNER_IMAGE"] = record.get("image", "marketplace-test-runner:lab2") if record and existing else runner_image()
        if record is None:
            # Persist ownership before Docker creation, including failures during preparation.
            save_json(self.metadata, {"stand": self.id, "project": self.project, "port": None, "image": self.env["RUNNER_IMAGE"]})

    def docker(self, *args, **kwargs):
        return subprocess.run(["docker", *args], cwd=ROOT, env=self.env, check=True, **kwargs)

    def compose(self, *args, **kwargs):
        return self.docker("compose", "-p", self.project, "-f", "docker-compose.yml", "-f", "docker-compose.test.yml", *args, **kwargs)

    @contextlib.contextmanager
    def lock(self, exclusive=True):
        with (self.directory / ".lock").open("a") as lock:
            try:
                fcntl.flock(lock, (fcntl.LOCK_EX if exclusive else fcntl.LOCK_SH) | fcntl.LOCK_NB)
            except BlockingIOError as error:
                raise RuntimeError(f"stand {self.id} is busy; exclusive and shared automation cannot overlap") from error
            self.refresh_ownership()
            yield

    def source(self):
        files = subprocess.check_output(["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"], cwd=ROOT)
        with tarfile.open(self.run / "source.tar", "w") as archive:
            for name in files.decode().split("\0"):
                if name and not name.startswith((".serena/", ".git/", "artifacts/")) and (ROOT / name).is_file():
                    archive.add(ROOT / name, arcname=name, recursive=False)
        save_json(self.run / "environment.json", {"commit": self.env["TEST_COMMIT_SHA"], "branch": subprocess.check_output(
            ["git", "branch", "--show-current"], cwd=ROOT, text=True).strip() or os.getenv("TEST_BRANCH", "detached"), "run": self.run_id,
            "stand": self.id, "working_tree_snapshot": not bool(self.env.get("TEST_REPOSITORY_URL")),
            "source_sha256": hashlib.sha256((self.run / "source.tar").read_bytes()).hexdigest(),
            "docker": self.docker("version", "--format", "{{.Server.Version}}", capture_output=True, text=True).stdout.strip()})

    def runner(self, stage):
        self.active = f"{self.project}-{self.run_id}-{stage}"
        print(f"[{self.run_id}] {stage}", flush=True)
        with (self.run / f"{stage}.log").open("wb") as output:
            try:
                self.compose("run", "--rm", "--no-deps", "--name", self.active, "runner", stage,
                             stdout=output, stderr=subprocess.STDOUT)
            finally:
                subprocess.run(["docker", "rm", "-f", self.active], env=self.env,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.active = None

    def ensure(self):
        self.env["RUNNER_IMAGE"] = runner_image()
        self.compose("config", "-q")
        self.compose("build", "runner", stdout=(self.run / "build.log").open("wb"), stderr=subprocess.STDOUT)
        self.compose("up", "-d", "--wait", "--wait-timeout", "90", "db", "cache")
        self.runner("prepare")

    def start_api(self):
        self.compose("up", "-d", "api")
        port = self.compose("port", "api", "8080", capture_output=True, text=True).stdout.strip().rsplit(":", 1)[1]
        save_json(self.metadata, {"stand": self.id, "project": self.project, "port": int(port), "image": self.env["RUNNER_IMAGE"]})
        self.ready()
        print(f"UI: http://127.0.0.1:{port} (STAND_ID={self.id})", flush=True)

    def ready(self):
        # Read-only: shared E2E must not recreate the server used by manual testers.
        self.compose("run", "--rm", "--no-deps", "--entrypoint", "bash", "runner", "-c",
                     'for i in {1..60}; do curl --fail --silent http://api:8080/api/v1/categories >/dev/null && exit 0; sleep 1; done; exit 1',
                     stdout=subprocess.DEVNULL)

    def db_dump(self, *options):
        return self.compose("exec", "-T", "db", "pg_dump", "-U", "postgres", *options, "marketplace", capture_output=True).stdout

    def snapshot(self, stages=None):
        self.compose("stop", "api")
        baseline = self.db_dump("--data-only", "--inserts")
        (self.run / "baseline.sql").write_bytes(baseline)
        (self.run / "baseline.dump").write_bytes(self.db_dump("-Fc"))
        schema = self.db_dump("--schema-only")
        save_json(self.run / "recovery.json", {"run": self.run_id, "stand": self.id, "project": self.project,
                  "created": self.created, "fingerprint": dump_fingerprint(baseline), "snapshot_ready": True,
                  "permissions_fingerprint": permissions_fingerprint(schema),
                  "restored": False, "stages": stages or ["unit", "integration", "e2e"]})
        self.publish()

    def restore(self):
        record = json.loads((self.run / "recovery.json").read_text())
        shared = record.get("mode") == "shared"
        if record["stand"] != self.id or record["project"] != self.project or (not shared and not record["snapshot_ready"]):
            raise ValueError("snapshot does not belong to this stand")
        if record.get("restored"):
            print(f"Recovery already completed for {self.run_id}; the stand is unchanged.", flush=True)
            return
        if not shared:
            self.compose("stop", "api")
        # Remove only this run's runner; the host remains available when it was SIGKILLed.
        for stage in ["prepare", "unit", "integration", "e2e", "traffic", "cleanup", "report"]:
            names = {f"{self.project}-{self.run_id}-{stage}", f"{self.project}-{self.run_id[:32]}-{stage}"}
            for name in names:
                subprocess.run(["docker", "rm", "-f", name], env=self.env,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.active = None
        if shared:
            self.ready()
            self.runner("cleanup")
            record["restored"] = True
            save_json(self.run / "recovery.json", record)
            return
        self.compose("exec", "-T", "db", "pg_restore", "--clean", "--if-exists",
                     "--exit-on-error", "-U", "postgres", "-d", "marketplace", input=(self.run / "baseline.dump").read_bytes())
        self.compose("exec", "-T", "cache", "redis-cli", "FLUSHDB", stdout=subprocess.DEVNULL)
        actual = self.db_dump("--data-only", "--inserts")
        (self.run / "restored.sql").write_bytes(actual)
        if dump_fingerprint(actual) != record["fingerprint"]:
            raise RuntimeError("restored data, sequences or migration state differ from baseline")
        expected_permissions = record.get("permissions_fingerprint")
        if "schema_fingerprint" in record:
            # Intermediate local snapshots used a format-sensitive full schema hash.
            archived_schema = self.compose("exec", "-T", "db", "pg_restore", "--schema-only", "-f", "-",
                                           input=(self.run / "baseline.dump").read_bytes(), capture_output=True).stdout
            expected_permissions = permissions_fingerprint(archived_schema)
        if expected_permissions and permissions_fingerprint(self.db_dump("--schema-only")) != expected_permissions:
            raise RuntimeError("restored ownership or privileges differ from baseline")
        self.start_api()
        self.runner("cleanup")
        record["restored"] = True
        save_json(self.run / "recovery.json", record)

    def logs(self):
        with (self.run / "server.log").open("wb") as output:
            self.compose("logs", "--no-color", "api", stdout=output, stderr=subprocess.STDOUT)

    def reports(self):
        self.runner("report")
        branch = json.loads((self.run / "environment.json").read_text())["branch"]
        history = ARTIFACTS / "history" / hashlib.sha256(branch.encode()).hexdigest()[:16]
        history.mkdir(parents=True, exist_ok=True)
        with (history / ".lock").open("a") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            previous = history / "history"
            if previous.exists():
                shutil.copytree(previous, self.run / "allure-results/history", dirs_exist_ok=True)
                self.runner("report")
            generated = self.run / "allure-report/history"
            if generated.exists():
                shutil.copytree(generated, previous, dirs_exist_ok=True)
            save_json(history / "latest.json", {"run": self.run_id, "branch": branch})

    def teardown(self):
        if not self.created:
            raise RuntimeError("refusing to delete a stand not created by this invocation")
        self.compose("down", "--volumes", "--remove-orphans")
        # Keep stand metadata and artifacts for diagnosis; mark it gone for future runs.
        if self.metadata.exists():
            self.metadata.unlink()

    def publish(self):
        self.artifacts.mkdir(parents=True, exist_ok=True)
        shutil.copytree(self.run, self.artifacts, dirs_exist_ok=True,
                        ignore=shutil.ignore_patterns("workspace", "*.credentials"))


def run(args):
    run_id = args.run_id or time.strftime("%Y%m%d-%H%M%S-") + uuid.uuid4().hex[:8]
    if args.command == "restore":
        if not args.run_id:
            raise ValueError("RUN_ID is required for restore")
        path = RUNTIME / "runs" / resource_id(run_id) / "recovery.json"
        if not path.exists():
            path = ARTIFACTS / run_id / "recovery.json"
        record = json.loads(path.read_text())
        if record.get("run") != run_id:
            raise ValueError("recovery record does not belong to RUN_ID")
        if record.get("restored"):
            print(f"Recovery already completed for {run_id}; the stand is unchanged.", flush=True)
            return 0
        stand = Stand(record["stand"], run_id, allow_existing=True)
        if not (stand.run / "recovery.json").exists():
            shutil.copytree(ARTIFACTS / run_id, stand.run, dirs_exist_ok=True)
        with stand.lock():
            require_recovered(record["stand"], restoring=run_id)
            stand.restore()
            status_file = stand.run / "stages.json"
            statuses = json.loads(status_file.read_text()) if status_file.exists() else {}
            for stage in record.get("stages", ["unit", "integration", "e2e"]):
                if statuses.get(stage) == "running":
                    statuses[stage] = "broken"
                    stage_result(stand.run / "allure-results", stage, "broken", "controller died; explicit recovery")
                elif stage not in statuses:
                    statuses[stage] = "skipped"
                    stage_result(stand.run / "allure-results", stage, "skipped", "controller died before stage")
            save_json(status_file, statuses)
            stand.reports()
            stand.publish()
        return 0
    stand_id = args.stand_id or "local"
    stand = Stand(stand_id, run_id)
    shared = args.command in ["e2e", "traffic", "guard-cleanup"]
    stages = ["unit", "integration", "e2e"] if args.command == "all" else [args.command]
    with stand.lock(exclusive=not shared):
        require_recovered(stand_id)
        success = False
        restore_ok = False
        snapshot_ready = False
        shared_ready = False
        try:
            stand.source()
            if shared:
                if stand.created:
                    raise RuntimeError("existing stand required; run make test-stand-up first")
                stand.ready()
                save_json(stand.run / "recovery.json", {"run": run_id, "stand": stand_id, "project": stand.project,
                          "mode": "shared", "snapshot_ready": False, "restored": False, "stages": stages})
                stand.publish()
                shared_ready = True
            else:
                stand.ensure()
            if args.command == "stand-up":
                stand.start_api()
                success = True
            else:
                if not shared:
                    print("Exclusive run: manual writes must be stopped until recovery completes.", flush=True)
                    stand.snapshot(stages)
                    snapshot_ready = True
                def execute(stage):
                    if stage in ["e2e", "traffic"] and not shared:
                        stand.start_api()
                    stand.runner(stage)
                statuses = pipeline(stand.run, stages, execute)
                success = all(status == "passed" for status in statuses.values())
        except BaseException as error:
            print(f"preparation: {error}", flush=True)
            stage_result(stand.run / "allure-results", "preparation", "broken", str(error))
            for stage in stages:
                stage_result(stand.run / "allure-results", stage, "skipped", "preparation failed")
        finally:
            operations = [("server-logs", stand.logs)]
            if shared:
                operations.append(("cleanup", stand.restore if shared_ready else lambda: stand.runner("cleanup")))
            elif snapshot_ready:
                operations.append(("restore", stand.restore))
            else:
                restore_ok = True  # No pre-test snapshot: no modifying test stage began.
            for name, operation in operations:
                try:
                    operation()
                    if name in ["cleanup", "restore"]:
                        restore_ok = True
                except BaseException as error:
                    print(f"{name}: {error}", flush=True)
                    stage_result(stand.run / "allure-results", name, "broken", str(error))
                    success = False
            try:
                stand.reports()
            except BaseException as error:
                print(f"report: {error}", flush=True)
                success = False
            if stand.created and restore_ok and not shared and args.command != "stand-up":
                stand.teardown()
            stand.publish()
        print(f"Artifacts: {stand.artifacts}", flush=True)
        return 0 if success and restore_ok else 1


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=["stand-up", "all", "integration", "e2e", "traffic", "restore", "guard-cleanup"])
    parser.add_argument("--stand-id", default=os.getenv("STAND_ID"))
    parser.add_argument("--run-id", default=os.getenv("RUN_ID"))
    def interrupted(signum, frame):
        raise KeyboardInterrupt(f"received signal {signum}")
    signal.signal(signal.SIGTERM, interrupted)
    try:
        raise SystemExit(run(parser.parse_args()))
    except (Exception, KeyboardInterrupt) as error:
        print(f"Test infrastructure failed: {error}", flush=True)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
