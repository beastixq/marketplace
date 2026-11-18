import tempfile
import unittest
import json
import os
import hashlib
import concurrent.futures
import threading
import io
import sys
from types import SimpleNamespace
from unittest.mock import patch
from pathlib import Path

from scripts.testing import controller


class ControllerTests(unittest.TestCase):
    def test_unit_preparation_does_not_start_or_migrate_application_services(self):
        with tempfile.TemporaryDirectory() as directory:
            stand = controller.Stand.__new__(controller.Stand)
            stand.run = Path(directory)
            stand.env = {}
            events = []
            stand.compose = lambda *args, **kwargs: events.append(args)
            stand.runner = lambda stage: events.append((stage,))
            stand.migrate = lambda: self.fail("unit must not migrate PostgreSQL")
            stand.prepare_tools()
            self.assertEqual(events, [("config", "-q"), ("prepare",)])

    def test_integration_preparation_starts_only_postgres(self):
        with tempfile.TemporaryDirectory() as directory:
            stand = controller.Stand.__new__(controller.Stand)
            stand.run = Path(directory)
            stand.env = {}
            events = []
            stand.prepare_tools = lambda: events.append(("tools",))
            stand.compose = lambda *args, **kwargs: events.append(args)
            stand.migrate = lambda: events.append(("migrate",))
            stand.ensure(database_only=True)
            self.assertEqual(events, [("tools",), ("up", "-d", "--wait", "--wait-timeout", "90", "db"), ("migrate",)])

    def test_common_stand_attaches_to_compose_without_an_ownership_record(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(controller, "RUNTIME", root), patch.object(controller, "ARTIFACTS", root / "artifacts"), patch.object(controller.Stand, "docker", return_value=SimpleNamespace(stdout="existing")), patch.object(controller.subprocess, "check_output", return_value="r1"):
                stand = controller.Stand("local", "r1")
                stand.source()
            self.assertEqual(stand.project, "marketplace")
            self.assertFalse(stand.created)
            self.assertFalse(stand.metadata.exists())
            self.assertEqual(stand.env.get("TEST_SOURCE_DIR"), str(controller.ROOT))
            self.assertFalse((stand.run / "source.tar").exists())

    def test_runner_streams_output_before_exit_and_preserves_logs_and_failure_code(self):
        for exit_code in (0, 7):
            with self.subTest(exit_code=exit_code), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                gate = root / "output-seen"
                stand = controller.Stand.__new__(controller.Stand)
                stand.run, stand.run_id, stand.project = root, "r1", "p1"
                stand.env = dict(os.environ, PATH=str(root) + os.pathsep + os.environ["PATH"],
                                 RUNNER_IMAGE="p1", OUTPUT_GATE=str(gate), EXIT_CODE=str(exit_code))
                executable = root / "docker"
                executable.write_text(f"#!{sys.executable}\n" +
                    "import os, sys, time\nfrom pathlib import Path\n"
                    "if sys.argv[1] != 'compose':\n raise SystemExit(0)\n"
                    "print('=== RUN TestP1', flush=True)\n"
                    "print('stderr: с1', end='', file=sys.stderr, flush=True)\n"
                    "gate = Path(os.environ['OUTPUT_GATE'])\n"
                    "deadline = time.monotonic() + 2\n"
                    "while not gate.exists() and time.monotonic() < deadline:\n time.sleep(0.01)\n"
                    "if not gate.exists():\n raise SystemExit(91)\n"
                    "print('\\nDONE', flush=True)\n"
                    "raise SystemExit(int(os.environ['EXIT_CODE']))\n")
                executable.chmod(0o700)

                class Terminal(io.StringIO):
                    def write(self, text):
                        written = super().write(text)
                        if "stderr: с1" in self.getvalue():
                            gate.touch()
                        return written

                terminal = Terminal()
                error = None
                with patch.object(sys, "stdout", terminal):
                    try:
                        stand.runner("integration")
                    except controller.subprocess.CalledProcessError as caught:
                        error = caught

                self.assertTrue(gate.exists(), "test output must reach the terminal before the process exits")
                expected = "=== RUN TestP1\nstderr: с1\nDONE\n"
                self.assertIn(expected, terminal.getvalue())
                self.assertEqual((root / "integration.log").read_text(), expected)
                if exit_code:
                    self.assertIsNotNone(error, "a failed test process must still fail the command")
                    self.assertEqual(error.returncode, exit_code)
                else:
                    self.assertIsNone(error)

    def test_host_migration_uses_working_sql_and_published_database_port(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            stand = controller.Stand.__new__(controller.Stand)
            stand.run, stand.run_id = root, "r1"
            stand.env = dict(os.environ, PATH=str(root) + os.pathsep + os.environ["PATH"],
                             MIGRATION_CALL=str(root / "call.json"), TEST_SOURCE_DIR=str(root))
            (root / "environment.json").write_text("{}")
            (root / "migrations").mkdir()
            (root / "migrations/001_create.sql").write_text("SELECT 1;")
            executable = root / "goose"
            executable.write_text(f"#!{sys.executable}\nimport json, os, sys\nfrom pathlib import Path\n"
                "if sys.argv[1:] == ['-version']:\n print('goose version: v3.26.0')\n"
                "else:\n Path(os.environ['MIGRATION_CALL']).write_text(json.dumps(sys.argv[1:]))\n")
            executable.chmod(0o700)
            self.assertTrue(callable(getattr(stand, "migrate", None)), "stand migration must invoke the host CLI")
            with patch.object(stand, "compose", return_value=SimpleNamespace(stdout="127.0.0.1:15433\n")):
                stand.migrate()
            args = json.loads((root / "call.json").read_text())
            self.assertEqual(args, ["-dir", str(root / "migrations"), "postgres",
                                    "postgres://postgres:postgres@127.0.0.1:15433/marketplace?sslmode=disable", "up"])
            self.assertEqual((root / "migrations/001_create.sql").read_text(), "SELECT 1;")
            self.assertFalse((root / "migrations/main.go").exists())
            self.assertEqual(json.loads((root / "environment.json").read_text())["goose"], "goose version: v3.26.0")

    def test_missing_host_goose_is_reported_before_connecting_to_docker(self):
        stand = controller.Stand.__new__(controller.Stand)
        stand.env = {"PATH": ""}
        self.assertTrue(callable(getattr(stand, "migrate", None)), "host CLI preflight is required")
        with patch.object(stand, "compose", side_effect=AssertionError("missing CLI must fail before Docker access")):
            with self.assertRaisesRegex(RuntimeError, "Goose CLI"):
                stand.migrate()

    def test_api_image_identifies_revision_and_dirty_build_content(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ["Dockerfile", ".dockerignore", "go.mod", "go.sum", "cmd/api/main.go", "internal/web/handler.go"]:
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(name)
            self.assertTrue(callable(getattr(controller, "api_image", None)), "API requires its own source-versioned image")
            revision = "0123456789abcdef0123456789abcdef01234567"
            clean, _ = controller.api_image(root, revision, "TAD-Lab2", False)
            self.assertEqual(clean, "marketplace-api:TAD-Lab2-0123456789ab")
            first, digest = controller.api_image(root, revision, "TAD-Lab2", True)
            self.assertRegex(first, r"^marketplace-api:TAD-Lab2-0123456789ab-dirty-[a-f0-9]{12}$")
            self.assertEqual(len(digest), 64)
            os.utime(root / "cmd/api/main.go", (1, 1))
            (root / "README.md").write_text("Documentation is not an API build input")
            self.assertEqual(controller.api_image(root, revision, "TAD-Lab2", True), (first, digest))
            (root / "internal/web/handler.go").write_text("changed application source")
            changed, changed_digest = controller.api_image(root, revision, "TAD-Lab2", True)
            self.assertNotEqual(changed, first)
            self.assertNotEqual(changed_digest, digest)
            branch, _ = controller.api_image(root, revision, "feature/" + "a" * 200, False)
            self.assertNotIn("/", branch)
            self.assertLessEqual(len(branch.split(":", 1)[1]), 128)

    def test_preparation_builds_api_from_test_workspace_and_keeps_old_version_on_failure(self):
        for fail in [False, True]:
            with self.subTest(fail=fail), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                with patch.object(controller, "RUNTIME", root), patch.object(controller, "ARTIFACTS", root / "artifacts"), patch.object(controller.Stand, "docker", return_value=SimpleNamespace(stdout="")), patch.object(controller.subprocess, "check_output", return_value="0123456789abcdef\n"):
                    stand = controller.Stand("dev", "build")
                    stand.env["API_SOURCE_DIR"] = str(stand.run / "workspace")
                old = json.loads(stand.metadata.read_text())
                old["api_image"] = "marketplace-api:previous"
                stand.metadata.write_text(json.dumps(old))
                (stand.run / "environment.json").write_text(json.dumps({
                    "commit": "0123456789abcdef", "branch": "TAD-Lab2", "working_tree_dirty": True,
                }))
                def prepare(stage):
                    self.assertEqual(stage, "prepare")
                    for name in ["Dockerfile", ".dockerignore", "go.mod", "go.sum", "cmd/api/main.go"]:
                        path = stand.run / "workspace" / name
                        path.parent.mkdir(parents=True, exist_ok=True)
                        path.write_text(name)
                def compose(*args, **kwargs):
                    if args == ("build", "api"):
                        if fail:
                            raise controller.subprocess.CalledProcessError(1, "docker build api")
                        self.assertEqual(stand.env["API_SOURCE_DIR"], str(stand.run / "workspace"))
                with patch.object(stand, "runner", side_effect=prepare), patch.object(stand, "compose", side_effect=compose), patch.object(stand, "migrate", create=True):
                    if fail:
                        with self.assertRaises(controller.subprocess.CalledProcessError):
                            stand.ensure()
                    else:
                        stand.ensure()
                actual = json.loads(stand.metadata.read_text())
                if fail:
                    self.assertEqual(actual["api_image"], "marketplace-api:previous")
                else:
                    self.assertTrue(actual["api_image"].startswith("marketplace-api:TAD-Lab2-0123456789ab-dirty-"))
                    self.assertNotEqual(actual["api_image"], actual["image"])
                    self.assertEqual(actual["api_revision"], "0123456789abcdef")
                    self.assertTrue(actual["api_dirty"])

    def test_start_api_preserves_build_identity_and_does_not_start_dependencies(self):
        with tempfile.TemporaryDirectory() as directory:
            stand = controller.Stand.__new__(controller.Stand)
            stand.id, stand.project = "dev", "marketplace-test-dev"
            stand.metadata = Path(directory) / "stand.json"
            stand.env = {"RUNNER_IMAGE": "marketplace-test-runner:tools", "API_IMAGE": "marketplace-api:revision"}
            stand.metadata.write_text(json.dumps({"api_image": "marketplace-api:revision", "api_revision": "0123456789abcdef"}))
            with patch.object(stand, "ready"), patch.object(stand, "compose", return_value=SimpleNamespace(stdout="127.0.0.1:32768\n")) as compose:
                stand.start_api()
            actual = json.loads(stand.metadata.read_text())
            self.assertEqual(actual.get("api_image"), "marketplace-api:revision")
            self.assertEqual(actual.get("api_revision"), "0123456789abcdef")
            self.assertEqual(actual["port"], 32768)
            self.assertEqual(compose.call_args_list[0].args, ("up", "-d", "--no-deps", "api"))

    def test_common_names_and_ports_are_fixed_while_isolated_stands_remain_independent(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(controller, "RUNTIME", root), patch.object(controller, "ARTIFACTS", root / "artifacts"), patch.object(controller.Stand, "docker", return_value=SimpleNamespace(stdout="")), patch.object(controller.subprocess, "check_output", return_value="0123456789abcdef\n"):
                common = controller.Stand("local", "r1")
                common.metadata.write_text(json.dumps({"port": 32768}))
                common.refresh_ownership()
                first = controller.Stand("isolated-r1", "r2")
                second = controller.Stand("isolated-r2", "r3")

            self.assertIsNone(common.env.get("UI_PORT"), "Compose owns the common stand's default ports")
            self.assertIsNone(common.env.get("DB_PORT"))
            self.assertIsNone(common.env.get("CACHE_PORT"))
            for service in ["api", "db", "cache"]:
                key = service.upper() + "_CONTAINER_NAME"
                self.assertIsNone(common.env.get(key), "Compose owns common container names")
                self.assertEqual(first.env.get(key), first.project + "-" + service + "-1")
                self.assertNotEqual(first.env.get(key), second.env.get(key))
            for stand in [first, second]:
                self.assertEqual(stand.env.get("UI_PORT"), "0")
                self.assertEqual(stand.env.get("CACHE_PORT"), "0")

    def test_isolated_runs_generate_unique_stands_and_remove_only_their_resources(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            stands = {}

            class FakeStand:
                created = True
                def __init__(self, stand_id, run_id):
                    self.run = root / run_id
                    self.run.mkdir()
                    self.artifacts = self.run
                    self.env = dict(os.environ)
                    self.events = stands[stand_id] = []
                def lock(self): return controller.contextlib.nullcontext()
                def source(self): pass
                def ensure(self, **kwargs):
                    if self.env.get("UI_PORT") != "0":
                        raise AssertionError("isolated runs must not inherit a fixed common-stand port")
                    if self.env.get("DB_PORT") != "0":
                        raise AssertionError("isolated runs must not inherit the common database port")
                    self.events.append("prepare")
                def snapshot(self, stages, **kwargs): self.events.append("snapshot")
                def start_api(self): self.events.append("start-api")
                def runner(self, stage): self.events.append(stage)
                def logs(self): pass
                def restore(self): self.events.append("restore")
                def reports(self): pass
                def publish(self): pass
                def teardown(self): self.events.append("teardown")

            commands = ["integration-isolated", "e2e-isolated", "all-isolated", "integration-isolated", "e2e-isolated"]
            def execute(pair):
                number, command = pair
                return controller.run(SimpleNamespace(command=command, run_id=f"isolated-{number}"))

            with patch.object(controller, "Stand", FakeStand), patch.object(controller, "require_recovered"), patch.dict(os.environ, STAND_ID="local", UI_PORT="9876", DB_PORT="5433"):
                with concurrent.futures.ThreadPoolExecutor(max_workers=5) as pool:
                    results = list(pool.map(execute, enumerate(commands)))
            self.assertEqual(results, [0] * 5)
            self.assertEqual(len(stands), 5, "parallel isolated invocations must not share resources")
            self.assertNotIn("local", stands)
            for events in stands.values():
                self.assertEqual(events[:2], ["prepare", "snapshot"])
                self.assertEqual(events[-2:], ["restore", "teardown"])
            self.assertEqual(json.loads((root / "isolated-2/stages.json").read_text()),
                             {"unit": "passed", "integration": "passed", "e2e": "passed"})

    def test_different_runner_build_inputs_do_not_share_a_mutable_image_tag(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "tests").mkdir()
            for name in ["Dockerfile", "requirements.txt", "runner.sh"]:
                (root / "tests" / name).write_text(name)
            with patch.object(controller, "ROOT", root):
                first = controller.runner_image()
                self.assertEqual(controller.runner_image(), first)
                (root / "tests/requirements.txt").write_text("different pinned dependencies")
                self.assertNotEqual(controller.runner_image(), first)

    def test_permissions_fingerprint_detects_grant_and_owner_loss_not_check_deparse(self):
        owner = b"ALTER TABLE public.products OWNER TO postgres;"
        grant = b"GRANT SELECT ON TABLE public.products TO marketplace_buyer;"
        expected = hashlib.sha256(owner + b"\n" + grant).hexdigest()
        for expression in [b"CHECK (role = ANY ((ARRAY['buyer'])::text[]))", b"CHECK (role = ANY (ARRAY['buyer'::text]))"]:
            schema = grant + b"\n" + expression + b"\n" + owner
            self.assertEqual(controller.permissions_fingerprint(schema), expected)
        self.assertNotEqual(controller.permissions_fingerprint(owner), expected)
        self.assertNotEqual(controller.permissions_fingerprint(grant), expected)

    def test_host_owns_allure_directory_before_a_container_can_create_it(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(controller, "RUNTIME", root), patch.object(controller, "ARTIFACTS", root / "artifacts"), patch.object(controller.Stand, "docker", return_value=SimpleNamespace(stdout="")), patch.object(controller.subprocess, "check_output", return_value="synthetic-commit\n"):
                stand = controller.Stand("shared", "first-e2e")
                results = stand.run / "allure-results"
                self.assertTrue(results.is_dir(), "host must create results before rootful pytest starts")
                self.assertEqual(results.stat().st_uid, os.geteuid())

    def test_resource_ownership_is_rechecked_after_acquiring_the_lock(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            existing = False
            deleted = root / "deleted"
            def docker(stand, *args, **kwargs):
                if args[0] == "compose":
                    deleted.touch()
                return SimpleNamespace(stdout="existing" if existing else "")
            with patch.object(controller, "RUNTIME", root), patch.object(controller, "ARTIFACTS", root / "artifacts"), patch.object(controller.Stand, "docker", docker), patch.object(controller.subprocess, "check_output", return_value="synthetic-commit\n"):
                stand = controller.Stand("race", "second")
                existing = True  # Another invocation created the stand before lock acquisition.
                with stand.lock(), self.assertRaisesRegex(RuntimeError, "not created"):
                    stand.teardown()
            self.assertFalse(deleted.exists())

    def test_restore_rechecks_completion_inside_the_stand_lock(self):
        with tempfile.TemporaryDirectory() as directory:
            stand = controller.Stand.__new__(controller.Stand)
            stand.id, stand.run_id, stand.project = "fake", "finished", "marketplace-test-fake"
            stand.run = Path(directory)
            (stand.run / "recovery.json").write_text(json.dumps({
                "run": stand.run_id, "stand": stand.id, "project": stand.project,
                "snapshot_ready": True, "restored": True,
            }))
            with patch.object(stand, "compose", side_effect=AssertionError("must not stop API or restore a completed run")):
                stand.restore()

    def test_unfinished_recovery_blocks_new_stand_mutations(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            previous = root / "runs" / "interrupted"
            previous.mkdir(parents=True)
            (previous / "recovery.json").write_text(json.dumps({
                "run": "interrupted", "stand": "fake", "snapshot_ready": True,
                "restored": False,
            }))
            changed = root / "new-baseline"
            class FakeStand:
                created = False
                def __init__(self, *args):
                    self.run = root / "new-run"
                    self.run.mkdir(exist_ok=True)
                    self.artifacts = self.run
                def lock(self, **kwargs): return controller.contextlib.nullcontext()
                def source(self): changed.touch()
                def ensure(self, **kwargs): pass
                def ready(self): pass
                def start_api(self): pass
                def snapshot(self, stages, **kwargs): pass
                def runner(self, stage): pass
                def logs(self): pass
                def restore(self): pass
                def reports(self): pass
                def publish(self): pass
            with patch.object(controller, "RUNTIME", root), patch.object(controller, "Stand", FakeStand):
                for command in ["all", "integration", "stand-up"]:
                    with self.subTest(command=command), self.assertRaisesRegex(RuntimeError, "interrupted"):
                        controller.run(SimpleNamespace(command=command, run_id="new"), owned_stand="fake", exclusive=True)
            self.assertFalse(changed.exists(), "blocked runs must not prepare a new baseline")

    def test_existing_integration_runs_tests_without_preparing_or_restoring_the_stand(self):
        for test_failure, report in [(False, False), (True, False), (False, True), (True, True)]:
            with self.subTest(test_failure=test_failure, report=report), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                events = []

                class FakeStand:
                    created = False
                    def __init__(self, stand_id, run_id):
                        if stand_id != "local":
                            raise AssertionError("public integration always targets the common local stand")
                        self.run = root
                        self.artifacts = root
                    def lock(self):
                        raise AssertionError("attached integration must not lock the stand")
                    def source(self): events.append("source")
                    def ensure(self): raise AssertionError("must not start dependencies or migrate")
                    def ready(self): raise AssertionError("must reach tests even when PostgreSQL is down")
                    def snapshot(self, stages): raise AssertionError("must not stop API or dump PostgreSQL")
                    def runner(self, stage):
                        events.append(stage)
                        if test_failure:
                            raise controller.subprocess.CalledProcessError(1, ["go", "test"])
                    def logs(self): events.append("logs")
                    def reports(self): events.append("report")
                    def publish(self): events.append("publish")
                    def restore(self): raise AssertionError("must not restore PostgreSQL or clear Redis")
                    def teardown(self): raise AssertionError("must not remove stand resources")

                with patch.object(controller, "Stand", FakeStand), patch.object(controller, "require_recovered", side_effect=AssertionError("must not wait for another run's recovery")), patch.dict(os.environ, STAND_ID="obsolete-other"):
                    try:
                        outcome = controller.run(SimpleNamespace(command="integration", run_id="attached", report=report))
                    except Exception as error:
                        outcome = str(error)
                self.assertEqual(outcome, 1 if test_failure else 0)
                self.assertEqual(events, ["source", "integration", "logs"] + (["report"] if report else []) + ["publish"])
                self.assertEqual(json.loads((root / "stages.json").read_text()),
                                 {"integration": "failed" if test_failure else "passed"})
                self.assertFalse((root / "recovery.json").exists(), "attached tests have no whole-DB recovery")

    def test_http_report_is_optional_without_changing_cleanup_or_test_status(self):
        for command, test_failure, report in [(command, failed, report) for command in ("e2e", "traffic")
                                              for failed in (False, True) for report in (False, True)]:
            with self.subTest(command=command, test_failure=test_failure, report=report), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                events = []

                class FakeStand:
                    created = False
                    project = "marketplace-test-local"
                    def __init__(self, *args):
                        self.run = root
                        self.artifacts = root
                    def source(self): pass
                    def runner(self, stage):
                        events.append(stage)
                        if test_failure:
                            raise controller.subprocess.CalledProcessError(1, ["pytest"])
                        if stage == "traffic":
                            (root / "purchase.pcap").write_bytes(b"pcap")
                    def logs(self): pass
                    def restore(self): events.append("cleanup")
                    def reports(self): events.append("report")
                    def publish(self): pass

                terminal = io.StringIO()
                with patch.object(controller, "Stand", FakeStand), patch.object(sys, "stdout", terminal):
                    outcome = controller.run(SimpleNamespace(command=command, run_id="r1", report=report))

                self.assertEqual(outcome, 1 if test_failure else 0)
                self.assertEqual(events, [command, "cleanup"] + (["report"] if report else []))
                self.assertEqual(json.loads((root / "stages.json").read_text()),
                                 {command: "failed" if test_failure else "passed"})
                if command == "traffic" and not test_failure:
                    self.assertIn(f"PCAP: {root / 'purchase.pcap'}", terminal.getvalue())

    def test_report_imports_previous_history_before_its_single_generation(self):
        for has_history in (False, True):
            with self.subTest(has_history=has_history), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                stand = controller.Stand.__new__(controller.Stand)
                stand.run = root / "run"
                stand.run.mkdir()
                stand.run_id = "r1"
                (stand.run / "environment.json").write_text(json.dumps({"branch": "b1"}))
                history = root / "history" / hashlib.sha256(b"b1").hexdigest()[:16] / "history"
                if has_history:
                    history.mkdir(parents=True)
                    (history / "history.json").write_text('{"previous": true}')
                generations = []

                def generate(stage):
                    imported = stand.run / "allure-results/history/history.json"
                    generations.append(imported.read_text() if imported.exists() else None)
                    target = stand.run / "allure-report/history"
                    target.mkdir(parents=True, exist_ok=True)
                    (target / "history.json").write_text('{"current": true}')

                stand.runner = generate
                with patch.object(controller, "ARTIFACTS", root):
                    stand.reports()
                self.assertEqual(generations, ['{"previous": true}' if has_history else None])
                self.assertEqual((history / "history.json").read_text(), '{"current": true}')

    def test_existing_integration_refuses_to_create_a_missing_stand(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            events = []

            class FakeStand:
                created = True
                def __init__(self, *args):
                    self.run = root
                    self.artifacts = root
                def lock(self): return controller.contextlib.nullcontext()
                def source(self): pass
                def ensure(self): events.append("unexpected setup")
                def snapshot(self, stages): events.append("unexpected snapshot")
                def runner(self, stage): events.append(stage)
                def logs(self): pass
                def reports(self): events.append("unexpected report container")
                def publish(self): pass
                def restore(self): events.append("unexpected recovery")
                def teardown(self): events.append("unexpected deletion")

            with patch.object(controller, "Stand", FakeStand), patch.object(controller, "require_recovered"):
                outcome = controller.run(SimpleNamespace(command="integration", run_id="attached"))
            self.assertEqual(outcome, 1)
            self.assertEqual(events, [])

    def test_shared_runs_do_not_wait_for_other_runs_or_unfinished_cleanup(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            previous = root / "runs" / "unfinished-cleanup"
            previous.mkdir(parents=True)
            (previous / "recovery.json").write_text(json.dumps({
                "run": "unfinished-cleanup", "stand": "local", "mode": "shared",
                "snapshot_ready": False, "restored": False,
            }))
            all_running = threading.Barrier(5)

            class FakeStand:
                created = False
                def __init__(self, stand_id, run_id):
                    self.run_id = run_id
                    self.project = "marketplace-test-" + stand_id
                    self.run = root / "runs" / run_id
                    self.run.mkdir()
                    self.artifacts = self.run
                def lock(self, **kwargs):
                    raise AssertionError("shared runs must not acquire the stand lock")
                def source(self): pass
                def ready(self): pass
                def publish(self): pass
                def logs(self): pass
                def reports(self): pass
                def runner(self, stage):
                    if stage == "e2e":
                        all_running.wait(timeout=5)
                def restore(self):
                    record = json.loads((self.run / "recovery.json").read_text())
                    record["restored"] = True
                    (self.run / "recovery.json").write_text(json.dumps(record))

            def execute(number):
                try:
                    return controller.run(SimpleNamespace(command="e2e", run_id=f"developer-{number}"))
                except Exception as error:
                    return str(error)

            with patch.object(controller, "RUNTIME", root), patch.object(controller, "ARTIFACTS", root / "artifacts"), patch.object(controller, "Stand", FakeStand):
                with concurrent.futures.ThreadPoolExecutor(max_workers=5) as pool:
                    outcomes = list(pool.map(execute, range(5)))
            self.assertEqual(outcomes, [0, 0, 0, 0, 0])
            for number in range(5):
                run = root / "runs" / f"developer-{number}"
                self.assertEqual(json.loads((run / "stages.json").read_text()), {"e2e": "passed"})
                self.assertTrue(json.loads((run / "recovery.json").read_text())["restored"])
            self.assertFalse(json.loads((previous / "recovery.json").read_text())["restored"])

    def test_unfinished_shared_cleanup_blocks_an_exclusive_baseline(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            previous = root / "runs" / "shared-interrupted"
            previous.mkdir(parents=True)
            (previous / "recovery.json").write_text(json.dumps({
                "run": "shared-interrupted", "stand": "shared", "mode": "shared",
                "snapshot_ready": False, "restored": False,
            }))
            with patch.object(controller, "RUNTIME", root), patch.object(controller, "ARTIFACTS", root / "artifacts"):
                with self.assertRaisesRegex(RuntimeError, "shared-interrupted"):
                    controller.require_recovered("shared")
                controller.require_recovered("shared", restoring="shared-interrupted")

    def test_shared_recovery_does_not_wait_for_other_unfinished_runs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for run_id in ["first", "second"]:
                run = root / "runs" / run_id
                run.mkdir(parents=True)
                (run / "recovery.json").write_text(json.dumps({
                    "run": run_id, "stand": "shared", "mode": "shared",
                    "snapshot_ready": False, "restored": False, "stages": ["e2e"],
                }))

            class FakeStand:
                def __init__(self, stand_id, run_id, allow_existing=False):
                    self.run = root / "runs" / run_id
                def lock(self):
                    raise AssertionError("addressed cleanup must not lock the shared stand")
                def restore(self):
                    path = self.run / "recovery.json"
                    record = json.loads(path.read_text())
                    record["restored"] = True
                    path.write_text(json.dumps(record))
                def reports(self): pass
                def publish(self): pass

            with patch.object(controller, "RUNTIME", root), patch.object(controller, "ARTIFACTS", root / "artifacts"), patch.object(controller, "Stand", FakeStand):
                try:
                    outcome = controller.run(SimpleNamespace(command="restore", run_id="first"))
                except Exception as error:
                    outcome = str(error)
            self.assertEqual(outcome, 0)
            self.assertTrue(json.loads((root / "runs/first/recovery.json").read_text())["restored"])
            self.assertFalse(json.loads((root / "runs/second/recovery.json").read_text())["restored"])

    def test_shared_restore_cleans_own_data_without_stopping_or_resetting_the_stand(self):
        with tempfile.TemporaryDirectory() as directory:
            stand = controller.Stand.__new__(controller.Stand)
            stand.id, stand.run_id, stand.project = "shared", "shared-interrupted", "marketplace-test-shared"
            stand.run, stand.env = Path(directory), {}
            (stand.run / "recovery.json").write_text(json.dumps({
                "run": stand.run_id, "stand": stand.id, "project": stand.project,
                "mode": "shared", "snapshot_ready": False, "restored": False,
            }))
            own, manual = stand.run / "own-fixture", stand.run / "manual-fixture"
            own.touch()
            manual.touch()
            def runner(stage):
                if stage != "cleanup": raise AssertionError(f"unexpected stage: {stage}")
                own.unlink()
            with patch.object(controller.subprocess, "run"), patch.object(stand, "ready", side_effect=AssertionError("shared cleanup must not wait for API")), patch.object(stand, "runner", runner), patch.object(stand, "compose", side_effect=AssertionError("shared recovery must not stop API, restore PostgreSQL or flush Redis")):
                stand.restore()
            self.assertFalse(own.exists())
            self.assertTrue(manual.exists())
            self.assertTrue(json.loads((stand.run / "recovery.json").read_text())["restored"])

    def test_completed_restore_is_a_non_mutating_noop(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            previous = root / "finished"
            previous.mkdir()
            (previous / "recovery.json").write_text(json.dumps({
                "run": "finished", "stand": "fake", "snapshot_ready": True,
                "restored": True,
            }))
            with patch.object(controller, "ARTIFACTS", root), patch.object(controller, "Stand", side_effect=AssertionError("must not touch the stand")):
                self.assertEqual(controller.run(SimpleNamespace(command="restore", run_id="finished")), 0)

    def test_restore_is_not_complete_until_readiness_and_cleanup_pass(self):
        with tempfile.TemporaryDirectory() as directory:
            stand = controller.Stand.__new__(controller.Stand)
            stand.id, stand.run_id, stand.project = "fake", "interrupted", "marketplace-test-fake"
            stand.run = Path(directory)
            stand.env = {}
            baseline = b"SELECT 1;\n"
            (stand.run / "baseline.dump").write_bytes(b"snapshot")
            record = {"run": stand.run_id, "stand": stand.id, "project": stand.project,
                      "snapshot_ready": True, "restored": False,
                      "fingerprint": controller.dump_fingerprint(baseline)}
            (stand.run / "recovery.json").write_text(json.dumps(record))
            for operation in ["start_api", "runner"]:
                (stand.run / "recovery.json").write_text(json.dumps(record))
                with self.subTest(operation=operation), patch.object(stand, "compose"), patch.object(stand, "db_dump", return_value=baseline), patch.object(controller.subprocess, "run"), patch.object(stand, "start_api", side_effect=RuntimeError("readiness") if operation == "start_api" else None), patch.object(stand, "runner", side_effect=RuntimeError("token check") if operation == "runner" else None):
                    with self.assertRaises(RuntimeError):
                        stand.restore()
                    self.assertFalse(json.loads((stand.run / "recovery.json").read_text())["restored"])

    def test_resource_ids_cannot_escape_artifact_directory(self):
        for value in ["../other", "/", "", "a;b", "a b"]:
            with self.assertRaises(ValueError):
                controller.resource_id(value)
        self.assertEqual(controller.resource_id("dev-1"), "dev-1")

    def test_fingerprint_ignores_dump_nonce_but_not_sequence_state(self):
        first = b"-- comment\n\\restrict abc\nSELECT setval('users_id_seq', 2, true);\n\\unrestrict abc\n"
        second = b"-- different\n\\restrict xyz\nSELECT setval('users_id_seq', 2, true);\n\\unrestrict xyz\n"
        self.assertEqual(controller.dump_fingerprint(first), controller.dump_fingerprint(second))
        self.assertNotEqual(controller.dump_fingerprint(first), controller.dump_fingerprint(second.replace(b"2, true", b"3, true")))

    def test_stage_failure_does_not_start_following_stages(self):
        called = []
        def execute(stage):
            called.append(stage)
            if stage == "unit":
                raise RuntimeError("test failed")
        with tempfile.TemporaryDirectory() as directory:
            statuses = controller.pipeline(Path(directory), ["unit", "integration", "e2e"], execute)
        self.assertEqual(called, ["unit"])
        self.assertEqual(statuses, {"unit": "broken", "integration": "skipped", "e2e": "skipped"})

    def test_preparation_failure_still_publishes_an_error_report(self):
        with tempfile.TemporaryDirectory() as directory:
            events = []
            class FakeStand:
                created = False
                def __init__(self, *args):
                    self.run = Path(directory)
                    self.artifacts = Path(directory)
                def lock(self, **kwargs):
                    return controller.contextlib.nullcontext()
                def source(self): events.append("source")
                def ensure(self): raise RuntimeError("prepare failed")
                def logs(self): events.append("logs")
                def reports(self): events.append("report")
                def publish(self): events.append("publish")
            with patch.object(controller, "Stand", FakeStand):
                result = controller.run(SimpleNamespace(command="all", run_id="fake"), owned_stand="fake", exclusive=True)
            self.assertEqual(result, 1)
            self.assertIn("report", events)
            self.assertIn("publish", events)

    def test_stand_up_does_not_generate_empty_test_reports(self):
        with tempfile.TemporaryDirectory() as directory:
            class FakeStand:
                created = True
                def __init__(self, *args):
                    self.run = Path(directory)
                    self.artifacts = Path(directory)
                def lock(self, **kwargs): return controller.contextlib.nullcontext()
                def source(self): pass
                def ensure(self): pass
                def start_api(self): pass
                def logs(self): pass
                def publish(self): pass
                def reports(self): raise RuntimeError("stand startup must not generate a test report")
            with patch.object(controller, "Stand", FakeStand), patch.object(controller, "require_recovered"):
                result = controller.run(SimpleNamespace(command="stand-up", run_id="fake"))
            self.assertEqual(result, 0)

    def test_existing_compose_project_requires_matching_ownership_record(self):
        for record in [None, {"stand": "other", "project": "marketplace-test-dev"},
                       {"stand": "dev", "project": "unrelated"}]:
            with self.assertRaises(ValueError):
                controller.check_ownership(record, "dev", "marketplace-test-dev", existing=True)
        controller.check_ownership({"stand": "dev", "project": "marketplace-test-dev"},
                                   "dev", "marketplace-test-dev", existing=True)

    def test_restore_failure_is_reported_and_does_not_delete_the_stand(self):
        with tempfile.TemporaryDirectory() as directory:
            events = []
            class FakeStand:
                created = True
                def __init__(self, *args):
                    self.run = Path(directory)
                    self.artifacts = Path(directory)
                    self.env = {}
                def lock(self, **kwargs): return controller.contextlib.nullcontext()
                def source(self): pass
                def ensure(self, **kwargs): pass
                def snapshot(self, stages, **kwargs): pass
                def runner(self, stage): events.append(stage)
                def logs(self): pass
                def restore(self): raise RuntimeError("pg_restore failed")
                def reports(self): events.append("report")
                def publish(self): events.append("publish")
                def teardown(self): events.append("unsafe deletion")
            with patch.object(controller, "Stand", FakeStand):
                result = controller.run(SimpleNamespace(command="integration-isolated", run_id="fake", report=True))
            self.assertEqual(result, 1)
            self.assertNotIn("unsafe deletion", events)
            self.assertIn("report", events)
            self.assertIn("publish", events)


if __name__ == "__main__":
    unittest.main()
