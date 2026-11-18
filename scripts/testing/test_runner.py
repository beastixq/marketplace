import tempfile
import unittest
import os
import subprocess
import signal
import json
from pathlib import Path
from unittest.mock import Mock, patch

from scripts.testing import runner
from scripts.testing.runner import verify_capture
from scripts.testing.controller import ROOT


class PreparationTests(unittest.TestCase):
    def test_prepare_preserves_explicit_stand_configuration(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            (output / "environment.json").write_text("{}")
            config = output / "config.yaml"
            config.write_text("user supplied configuration")
            with patch.object(runner, "OUTPUT", output), patch.object(runner.subprocess, "check_output", return_value="version"), patch.object(runner, "command"):
                runner.main("prepare")
            self.assertEqual(config.read_text(), "user supplied configuration")

    def test_prepare_does_not_migrate_database_or_build_an_external_api_binary(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            (output / "environment.json").write_text("{}")
            with patch.object(runner, "OUTPUT", output), patch.object(runner.subprocess, "check_output", return_value="version"), patch.object(runner, "command") as execute:
                runner.main("prepare")
            commands = [call.args[0] for call in execute.call_args_list]
            self.assertFalse(any(args[0] == "goose" for args in commands), "migrations belong to the host CLI, not the test container")
            self.assertFalse(any(args[:2] == ["go", "build"] for args in commands), "API must be built into its own image, not written to the stand directory")
            self.assertEqual(json.loads((output / "environment.json").read_text())["go"], "version")


class TrafficTests(unittest.TestCase):
    def test_traffic_capture_finishes_without_sending_http_probes(self):
        capture = Mock()
        capture.poll.side_effect = [None, None, 0, 0]
        capture.wait.return_value = 0
        with tempfile.TemporaryDirectory() as directory, patch.object(runner, "OUTPUT", Path(directory)), patch.object(runner.subprocess, "Popen", return_value=capture) as start, patch.object(runner.subprocess, "run") as execute, patch.object(runner, "verify_capture"), patch("time.sleep"):
            runner.main("traffic")
        requests = [call.args[0] for call in execute.call_args_list]
        self.assertFalse(any(args[0] == "curl" for args in requests), "capture shutdown must not send requests to the application")
        self.assertEqual(start.call_args.args[0][0], "tcpdump")
        self.assertIn("--immediate-mode", start.call_args.args[0])
        self.assertIn("-U", start.call_args.args[0])
        capture.send_signal.assert_called_once_with(signal.SIGINT)
        capture.wait.assert_called_once_with(timeout=20)

    def test_traffic_capture_failure_is_not_reported_as_success(self):
        capture = Mock()
        capture.poll.side_effect = [None, None, 0, 0]
        capture.wait.return_value = 1
        with tempfile.TemporaryDirectory() as directory, patch.object(runner, "OUTPUT", Path(directory)), patch.object(runner.subprocess, "Popen", return_value=capture), patch.object(runner.subprocess, "run"), patch.object(runner, "verify_capture"), patch("time.sleep"):
            with self.assertRaisesRegex(RuntimeError, "capture.*exit"):
                runner.main("traffic")

    def test_capture_signal_denial_explains_the_host_permission_problem(self):
        capture = Mock()
        capture.poll.return_value = None
        capture.send_signal.side_effect = PermissionError("signal denied")
        with tempfile.TemporaryDirectory() as directory, patch.object(runner, "OUTPUT", Path(directory)), patch.object(runner.subprocess, "Popen", return_value=capture), patch.object(runner.subprocess, "run"), patch("time.sleep"):
            with self.assertRaisesRegex(RuntimeError, "AppArmor"):
                runner.main("traffic")

    def test_capture_requires_paired_response_not_only_request_uri(self):
        requests = [("POST", "/api/v1/auth/login", 200), ("POST", "/api/v1/cart/items", 204),
                    ("POST", "/api/v1/orders", 201), ("POST", "/api/v1/orders/9/payment-link", 200),
                    ("POST", "/api/v1/payments/callback/mock-bank", 200),
                    ("POST", "/api/v1/orders/9/ship", 204), ("POST", "/api/v1/orders/9/deliver", 204),
                    ("POST", "/api/v1/reviews", 201), ("GET", "/api/v1/products/5/reviews?page=1&limit=20", 200)]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "capture.tsv"
            rows = []
            for number, (method, uri, code) in enumerate(requests, 1):
                rows.extend([f"{number}\t{method}\t{uri}\t\t", f"{number + 100}\t\t\t{code}\t{number}"])
            path.write_text("\n".join(rows))
            verify_capture(path)
            path.write_text("\n".join(rows[:-1]))
            with self.assertRaisesRegex(RuntimeError, "paired HTTP"):
                verify_capture(path)

    @unittest.skipUnless((ROOT / ".git").exists(), "local container uses a source snapshot, not a Git checkout")
    def test_runner_reuses_exact_commit_checkout_across_stages_with_foreign_owner(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory)
            workspace = target / "workspace"
            workspace.mkdir()
            executable = target / "python3"
            executable.write_text("#!/bin/sh\nexit 0\n")
            executable.chmod(0o700)
            commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
            env = dict(os.environ, PATH=str(target) + ":" + os.environ["PATH"], TEST_WORKSPACE=str(workspace),
                       TEST_REPOSITORY_URL=str(ROOT), TEST_COMMIT_SHA=commit,
                       GIT_TEST_ASSUME_DIFFERENT_OWNER="1", GIT_CONFIG_GLOBAL=str(target / "gitconfig"))
            for source in [ROOT, ROOT / ".git"]:
                subprocess.run(["git", "config", "--global", "--add", "safe.directory", str(source)],
                               env=env, check=True)
            for stage in ["prepare", "unit", "report"]:
                result = subprocess.run(["bash", str(ROOT / "tests/runner.sh"), stage], env=env,
                                        capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                actual = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=workspace,
                                                 env=env, text=True).strip()
                self.assertEqual(actual, commit)
            trusted = subprocess.check_output(["git", "config", "--global", "--get-all", "safe.directory"],
                                              env=env, text=True).splitlines()
            self.assertEqual(set(trusted), {str(ROOT), str(ROOT / ".git"), str(workspace)})


if __name__ == "__main__":
    unittest.main()
