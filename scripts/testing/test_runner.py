import tempfile
import unittest
import os
import subprocess
from pathlib import Path

from scripts.testing.runner import verify_capture
from scripts.testing.controller import ROOT


class TrafficTests(unittest.TestCase):
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
