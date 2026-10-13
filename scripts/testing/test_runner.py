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
    def test_runner_reuses_exact_commit_checkout_across_stages(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory)
            workspace = target / "workspace"
            workspace.mkdir()
            executable = target / "python3"
            executable.write_text("#!/bin/sh\nexit 0\n")
            executable.chmod(0o700)
            commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
            env = dict(os.environ, PATH=str(target) + ":" + os.environ["PATH"], TEST_WORKSPACE=str(workspace),
                       TEST_REPOSITORY_URL=str(ROOT), TEST_COMMIT_SHA=commit)
            for stage in ["prepare", "unit", "report"]:
                subprocess.run(["bash", str(ROOT / "tests/runner.sh"), stage], env=env,
                               check=True, capture_output=True)
                actual = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=workspace, text=True).strip()
                self.assertEqual(actual, commit)


if __name__ == "__main__":
    unittest.main()
