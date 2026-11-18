"""Host-side owner of Docker processes, snapshots and recovery (stdlib only)."""

import argparse
import codecs
import contextlib
import fcntl
import hashlib
import json
import os
import re
import shutil
import signal
import subprocess
import sys
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


def api_image(workspace, revision, branch, dirty):
    digest = hashlib.sha256()
    paths = [workspace / name for name in ["Dockerfile", ".dockerignore", "go.mod", "go.sum"]]
    for name in ["cmd/api", "internal"]:
        paths.extend(path for path in (workspace / name).rglob("*") if path.is_file())
    for path in sorted(paths):
        digest.update(path.relative_to(workspace).as_posix().encode() + b"\0" + path.read_bytes() + b"\0")
    source_hash = digest.hexdigest()
    branch = re.sub(r"[^A-Za-z0-9_.-]+", "-", branch).strip(".-")[:64] or "detached"
    tag = f"{branch}-{revision[:12]}"
    if dirty:
        tag += "-dirty-" + source_hash[:12]
    return "marketplace-api:" + tag, source_hash


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
        self.project = os.getenv("COMPOSE_PROJECT_NAME", "marketplace") if self.id == "local" else "marketplace-test-" + self.id
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
        self.env = dict(os.environ, STAND_PATH=str(self.directory), RUN_PATH=str(self.run), RUN_ID=self.run_id,
                        CACHE_PATH=str(cache))
        source = self.run / "workspace" if self.env.get("TEST_REPOSITORY_URL") else ROOT
        self.env.update(TEST_SOURCE_DIR=str(source), API_SOURCE_DIR=str(source))
        if self.id != "local":
            self.env.update(DB_PORT="0", UI_PORT="0", CACHE_PORT="0")
            for service in ["api", "db", "cache"]:
                self.env[service.upper() + "_CONTAINER_NAME"] = self.project + "-" + service + "-1"
        self.env.setdefault("BUILDX_CONFIG", "/tmp/marketplace-buildx")
        self.env["TEST_COMMIT_SHA"] = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
        self.metadata = self.directory / "stand.json"
        self.active = None
        self.refresh_ownership()

    def refresh_ownership(self):
        record = json.loads(self.metadata.read_text()) if self.id != "local" and self.metadata.exists() else None
        if self.id != "local" and record and record.get("port"):
            self.env["UI_PORT"] = str(record["port"])
        selector = "label=com.docker.compose.project=" + self.project
        existing = any(self.docker(*command, "--filter", selector, "--format", format,
                                  capture_output=True, text=True).stdout.strip()
                       for command, format in [(["ps", "-a"], "{{.ID}}"), (["network", "ls"], "{{.ID}}"),
                                               (["volume", "ls"], "{{.Name}}")])
        self.created = not existing
        if self.id == "local":
            self.env["RUNNER_IMAGE"] = runner_image()
            self.env.setdefault("API_IMAGE", "marketplace-api:local")
            return
        check_ownership(record, self.id, self.project, existing)
        self.env["RUNNER_IMAGE"] = record.get("image", "marketplace-test-runner:lab2") if record and existing else runner_image()
        self.env["API_IMAGE"] = record.get("api_image", "marketplace-api:unbuilt") if record else "marketplace-api:unbuilt"
        self.env["API_REVISION"] = record.get("api_revision", "unknown") if record else "unknown"
        if record is None:
            # Persist ownership before Docker creation, including failures during preparation.
            save_json(self.metadata, {"stand": self.id, "project": self.project, "port": None, "image": self.env["RUNNER_IMAGE"]})

    def docker(self, *args, log=None, **kwargs):
        command = ["docker", *args]
        if log is None:
            return subprocess.run(command, cwd=ROOT, env=self.env, check=True, **kwargs)
        with subprocess.Popen(command, cwd=ROOT, env=self.env, stdout=subprocess.PIPE,
                              stderr=subprocess.STDOUT, **kwargs) as process:
            decoder = codecs.getincrementaldecoder("utf-8")(errors="replace")
            while chunk := process.stdout.read1(65536):
                log.write(chunk)
                log.flush()
                sys.stdout.write(decoder.decode(chunk))
                sys.stdout.flush()
            sys.stdout.write(decoder.decode(b"", final=True))
            sys.stdout.flush()
            returncode = process.wait()
        if returncode:
            raise subprocess.CalledProcessError(returncode, command)
        return subprocess.CompletedProcess(command, returncode)

    def compose(self, *args, **kwargs):
        return self.docker("compose", "-p", self.project, "-f", "docker-compose.yml", "-f", "docker-compose.test.yml", *args, **kwargs)

    @contextlib.contextmanager
    def lock(self):
        with (self.directory / ".lock").open("a") as lock:
            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError as error:
                raise RuntimeError(f"stand {self.id} is busy with another exclusive command") from error
            self.refresh_ownership()
            yield

    def source(self):
        dirty = not self.env.get("TEST_REPOSITORY_URL") and bool(subprocess.check_output(
            ["git", "status", "--porcelain", "--untracked-files=normal", "--", ".", ":(exclude).serena", ":(exclude)artifacts"],
            cwd=ROOT, text=True).strip())
        save_json(self.run / "environment.json", {"commit": self.env["TEST_COMMIT_SHA"], "branch": subprocess.check_output(
            ["git", "branch", "--show-current"], cwd=ROOT, text=True).strip() or os.getenv("TEST_BRANCH", "detached"), "run": self.run_id,
            "stand": self.id, "source_mode": "git-checkout" if self.env.get("TEST_REPOSITORY_URL") else "working-tree",
            "working_tree_dirty": bool(dirty), "runner_image": self.env["RUNNER_IMAGE"],
            "api_image": self.env["API_IMAGE"], "api_revision": self.env.get("API_REVISION", "unknown"),
            "docker": self.docker("version", "--format", "{{.Server.Version}}", capture_output=True, text=True).stdout.strip()})

    def runner(self, stage):
        self.active = f"{self.project}-{self.run_id}-{stage}"
        print(f"[{self.run_id}] {stage}", flush=True)
        with (self.run / f"{stage}.log").open("wb") as output:
            try:
                present = subprocess.run(["docker", "image", "inspect", self.env["RUNNER_IMAGE"]],
                                         env=self.env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                if present.returncode:
                    self.compose("build", "runner", log=output)
                self.compose("run", "--rm", "--no-deps", "--pull", "never", "--name", self.active, "runner", stage,
                             log=output)
            finally:
                subprocess.run(["docker", "rm", "-f", self.active], env=self.env,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.active = None

    def prepare_tools(self):
        self.env["RUNNER_IMAGE"] = runner_image()
        self.compose("config", "-q")
        self.runner("prepare")

    def ensure(self, *, database_only=False):
        self.prepare_tools()
        services = ["db"] if database_only else ["db", "cache"]
        self.compose("up", "-d", "--wait", "--wait-timeout", "90", *services)
        self.migrate()
        if database_only:
            return
        environment = json.loads((self.run / "environment.json").read_text())
        dirty = environment["working_tree_dirty"]
        image, source_hash = api_image(Path(self.env["API_SOURCE_DIR"]), environment["commit"], environment["branch"], dirty)
        self.env.update(API_IMAGE=image, API_SOURCE_SHA256=source_hash, API_DIRTY=str(dirty).lower())
        with (self.run / "api-build.log").open("wb") as output:
            self.compose("build", "api", stdout=output, stderr=subprocess.STDOUT)
        metadata = json.loads(self.metadata.read_text()) if self.metadata.exists() else {}
        metadata.update(image=self.env["RUNNER_IMAGE"], api_image=image, api_revision=environment["commit"],
                        api_source_sha256=source_hash, api_dirty=dirty)
        if self.id != "local":
            save_json(self.metadata, metadata)
        environment.update(runner_image=self.env["RUNNER_IMAGE"], api_image=image, api_revision=environment["commit"],
                           api_source_sha256=source_hash, api_dirty=dirty)
        save_json(self.run / "environment.json", environment)

    def migrate(self):
        goose = shutil.which("goose", path=self.env.get("PATH"))
        if not goose:
            raise RuntimeError("Goose CLI is required on the host; install goose and put it in PATH")
        migrations = Path(self.env["TEST_SOURCE_DIR"]) / "migrations"
        if not any(migrations.glob("*.sql")):
            raise RuntimeError("source directory contains no SQL migrations")
        port = self.compose("port", "db", "5432", capture_output=True, text=True).stdout.strip().rsplit(":", 1)[1]
        database_url = f"postgres://postgres:postgres@127.0.0.1:{port}/marketplace?sslmode=disable"
        print(f"[{self.run_id}] migrate (host Goose CLI)", flush=True)
        with (self.run / "migrate.log").open("wb") as output:
            subprocess.run(["make", "migrate", f"MIGRATIONS_DIR={migrations}"], cwd=ROOT,
                           env=dict(self.env, DATABASE_URL=database_url, GOOSE=goose), check=True,
                           stdout=output, stderr=subprocess.STDOUT)
        environment = json.loads((self.run / "environment.json").read_text())
        environment["goose"] = subprocess.check_output([goose, "-version"], env=self.env, text=True).strip()
        environment["database_host_port"] = int(port)
        environment["migration"] = self.compose("exec", "-T", "db", "psql", "-U", "postgres", "-d", "marketplace", "-Atc",
                                               "SELECT max(version_id) FROM goose_db_version WHERE is_applied", capture_output=True, text=True).stdout.strip()
        save_json(self.run / "environment.json", environment)

    def start_api(self):
        self.compose("up", "-d", "--no-deps", "api")
        port = self.compose("port", "api", "8080", capture_output=True, text=True).stdout.strip().rsplit(":", 1)[1]
        if self.id != "local":
            metadata = json.loads(self.metadata.read_text())
            metadata.update(stand=self.id, project=self.project, port=int(port), image=self.env["RUNNER_IMAGE"])
            save_json(self.metadata, metadata)
        self.ready()
        print(f"UI: http://127.0.0.1:{port}", flush=True)

    def ready(self):
        # Read-only: shared E2E must not recreate the server used by manual testers.
        self.docker("image", "inspect", "--format", "{{.Id}}", self.env["RUNNER_IMAGE"], stdout=subprocess.DEVNULL)
        self.compose("run", "--rm", "--no-deps", "--pull", "never", "--entrypoint", "bash", "runner", "-c",
                     'for i in {1..30}; do curl --connect-timeout 1 --max-time 2 --fail --silent http://api:8080/api/v1/categories >/dev/null && exit 0; sleep 1; done; exit 1',
                     stdout=subprocess.DEVNULL)

    def db_dump(self, *options):
        return self.compose("exec", "-T", "db", "pg_dump", "-U", "postgres", *options, "marketplace", capture_output=True).stdout

    def snapshot(self, stages=None, *, services=None):
        self.compose("stop", "api")
        baseline = self.db_dump("--data-only", "--inserts")
        (self.run / "baseline.sql").write_bytes(baseline)
        (self.run / "baseline.dump").write_bytes(self.db_dump("-Fc"))
        schema = self.db_dump("--schema-only")
        save_json(self.run / "recovery.json", {"run": self.run_id, "stand": self.id, "project": self.project,
                  "created": self.created, "fingerprint": dump_fingerprint(baseline), "snapshot_ready": True,
                  "permissions_fingerprint": permissions_fingerprint(schema),
                  "services": services or ["db", "cache", "api"],
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
            self.runner("cleanup")
            record["restored"] = True
            save_json(self.run / "recovery.json", record)
            return
        self.compose("exec", "-T", "db", "pg_restore", "--clean", "--if-exists",
                     "--exit-on-error", "-U", "postgres", "-d", "marketplace", input=(self.run / "baseline.dump").read_bytes())
        services = record.get("services", ["db", "cache", "api"])
        if "cache" in services:
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
        if "api" in services:
            self.start_api()
            self.runner("cleanup")
        record["restored"] = True
        save_json(self.run / "recovery.json", record)

    def logs(self):
        with (self.run / "server.log").open("wb") as output:
            self.compose("logs", "--no-color", "api", stdout=output, stderr=subprocess.STDOUT)

    def reports(self):
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


def run(args, *, owned_stand=None, exclusive=False):
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
        shared = record.get("mode") == "shared"
        scope = contextlib.nullcontext() if shared else stand.lock()
        with scope:
            if not shared:
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
    isolated = args.command.endswith("-isolated")
    command = args.command.removesuffix("-isolated")
    report_enabled = getattr(args, "report", False) or command not in ["unit", "integration", "e2e", "traffic"]
    exclusive = exclusive or isolated
    stand_id = owned_stand or ("isolated-" + uuid.uuid4().hex[:16] if isolated else "local")
    stand = Stand(stand_id, run_id)
    if isolated and not stand.created:
        raise RuntimeError("isolated resource name already exists; refusing reuse, retry the command")
    if isolated:
        stand.env["UI_PORT"] = "0"
        stand.env["DB_PORT"] = "0"
    attached = not exclusive and command != "stand-up"
    stages = ["unit", "integration", "e2e"] if command == "all" else [command]
    scope = contextlib.nullcontext() if attached else stand.lock()
    with scope:
        if not attached:
            require_recovered(stand_id)
        success = False
        restore_ok = False
        snapshot_ready = False
        shared_ready = False
        try:
            stand.source()
            if attached and stand.created:
                raise RuntimeError("existing stand required; run make test-stand-up first")
            if not attached:
                if command == "unit":
                    stand.prepare_tools()
                elif command == "integration" and isolated:
                    stand.ensure(database_only=True)
                else:
                    stand.ensure()
            if command == "stand-up":
                stand.start_api()
                success = True
            else:
                if not attached and command != "unit":
                    print("Owned temporary environment: snapshot/recovery does not touch the common stand.", flush=True)
                    if command == "integration" and isolated:
                        stand.snapshot(stages, services=["db"])
                    else:
                        stand.snapshot(stages)
                    snapshot_ready = True
                elif command in ["integration", "all"]:
                    print("Existing-stand integration: dependencies are unchanged; no snapshot or recovery. "
                          "Tests can modify shared orders (ExpireOrders); avoid concurrent manual writes.", flush=True)
                def execute(stage):
                    nonlocal shared_ready
                    if stage in ["e2e", "traffic", "guard-cleanup"]:
                        if not attached:
                            stand.start_api()
                        elif not shared_ready:
                            save_json(stand.run / "recovery.json", {"run": run_id, "stand": stand_id, "project": stand.project,
                                      "mode": "shared", "snapshot_ready": False, "restored": False, "stages": stages})
                            shared_ready = True
                            stand.publish()
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
            if shared_ready:
                operations.append(("cleanup", stand.restore))
            elif snapshot_ready:
                operations.append(("restore", stand.restore))
            else:
                restore_ok = True  # No snapshot belongs to this path; fixtures own their cleanup.
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
                if report_enabled and command != "stand-up" and (not attached or not stand.created):
                    stand.reports()
            except BaseException as error:
                print(f"report: {error}", flush=True)
                success = False
            if stand.created and restore_ok and not attached and command != "stand-up":
                stand.teardown()
            stand.publish()
        print(f"Artifacts: {stand.artifacts}", flush=True)
        if command == "traffic" and (stand.artifacts / "purchase.pcap").exists():
            print(f"PCAP: {stand.artifacts / 'purchase.pcap'}", flush=True)
        return 0 if success and restore_ok else 1


def main(*, owned_stand=None, exclusive=False):
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=["stand-up", "all", "integration", "e2e", "traffic", "restore", "guard-cleanup",
                                            "unit-isolated", "all-isolated", "integration-isolated", "e2e-isolated", "traffic-isolated"])
    parser.add_argument("--run-id", default=os.getenv("RUN_ID"))
    parser.add_argument("--report", action="store_true", help="generate Allure HTML after integration/E2E/traffic tests")
    def interrupted(signum, frame):
        raise KeyboardInterrupt(f"received signal {signum}")
    signal.signal(signal.SIGTERM, interrupted)
    try:
        raise SystemExit(run(parser.parse_args(), owned_stand=owned_stand, exclusive=exclusive))
    except (Exception, KeyboardInterrupt) as error:
        print(f"Test infrastructure failed: {error}", flush=True)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
