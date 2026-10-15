import tempfile
import unittest
import json
import os
import hashlib
from types import SimpleNamespace
from unittest.mock import patch
from pathlib import Path

from scripts.testing import controller


class ControllerTests(unittest.TestCase):
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
                def ensure(self): pass
                def ready(self): pass
                def start_api(self): pass
                def snapshot(self, stages): pass
                def runner(self, stage): pass
                def logs(self): pass
                def restore(self): pass
                def reports(self): pass
                def publish(self): pass
            with patch.object(controller, "RUNTIME", root), patch.object(controller, "Stand", FakeStand):
                for command in ["all", "integration", "stand-up", "e2e", "traffic"]:
                    with self.subTest(command=command), self.assertRaisesRegex(RuntimeError, "interrupted"):
                        controller.run(SimpleNamespace(command=command, stand_id="fake", run_id="new"))
            self.assertFalse(changed.exists(), "blocked runs must not prepare a new baseline")

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
            with patch.object(controller.subprocess, "run"), patch.object(stand, "ready"), patch.object(stand, "runner", runner), patch.object(stand, "compose", side_effect=AssertionError("shared recovery must not stop API, restore PostgreSQL or flush Redis")):
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
                self.assertEqual(controller.run(SimpleNamespace(command="restore", stand_id=None, run_id="finished")), 0)

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
                result = controller.run(SimpleNamespace(command="all", stand_id="fake", run_id="fake"))
            self.assertEqual(result, 1)
            self.assertIn("report", events)
            self.assertIn("publish", events)

    def test_stand_up_does_not_hide_report_failure(self):
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
                def reports(self): raise RuntimeError("report failed")
            with patch.object(controller, "Stand", FakeStand):
                result = controller.run(SimpleNamespace(command="stand-up", stand_id="fake", run_id="fake"))
            self.assertEqual(result, 1)

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
                def lock(self, **kwargs): return controller.contextlib.nullcontext()
                def source(self): pass
                def ensure(self): pass
                def snapshot(self, stages): pass
                def runner(self, stage): events.append(stage)
                def logs(self): pass
                def restore(self): raise RuntimeError("pg_restore failed")
                def reports(self): events.append("report")
                def publish(self): events.append("publish")
                def teardown(self): events.append("unsafe deletion")
            with patch.object(controller, "Stand", FakeStand):
                result = controller.run(SimpleNamespace(command="integration", stand_id="fake", run_id="fake"))
            self.assertEqual(result, 1)
            self.assertNotIn("unsafe deletion", events)
            self.assertIn("report", events)
            self.assertIn("publish", events)


if __name__ == "__main__":
    unittest.main()
