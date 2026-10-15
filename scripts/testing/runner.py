"""Commands inside the test container; database lifecycle belongs to the host."""

import os
import csv
import re
import shutil
import subprocess
import sys
from pathlib import Path

from scripts.testing.results import junit_to_allure, public_results, require_integration

OUTPUT = Path("/run")
PACKAGES = ["./internal/service", "./internal/repository", "./internal/config",
            "./internal/logging", "./internal/middleware", "./internal/web"]


def command(args, **kwargs):
    subprocess.run(args, check=True, **kwargs)


def go_tests(stage, args):
    raw = OUTPUT / "raw"
    raw.mkdir(exist_ok=True)
    command(["go", "tool", "gotestsum", "--format", "pkgname-and-test-fails",
             "--junitfile", str(raw / f"{stage}.xml"), "--jsonfile", str(raw / f"{stage}.jsonl"), "--", *args])
    if stage == "integration":
        require_integration(raw / "integration.jsonl", ["github.com/beastixq/marketplace/internal/repository",
                                                       "github.com/beastixq/marketplace/internal/service"])


def verify_capture(path):
    with Path(path).open() as source:
        rows = list(csv.reader(source, delimiter="\t"))
    responses = {row[4]: row[3] for row in rows if len(row) == 5 and row[4] and row[3]}
    required = [("POST", r"/auth/login", 200), ("POST", r"/cart/items", 204), ("POST", r"/orders", 201),
                ("POST", r"/orders/\d+/payment-link", 200), ("POST", r"/payments/callback/mock-bank", 200),
                ("POST", r"/orders/\d+/ship", 204), ("POST", r"/orders/\d+/deliver", 204),
                ("POST", r"/reviews", 201), ("GET", r"/products/\d+/reviews\?page=1&limit=20", 200)]
    for method, uri, code in required:
        if not any(len(row) == 5 and row[1] == method and re.fullmatch("/api/v1" + uri, row[2]) and
                   responses.get(row[0]) == str(code) for row in rows):
            raise RuntimeError(f"capture misses paired HTTP {code}: {method} {uri}")


def main(stage):
    if stage == "prepare":
        command(["go", "mod", "download"])
        command(["goose", "-dir", "migrations", "postgres", os.environ["DATABASE_URL"], "up"])
        import json
        environment = json.loads((OUTPUT / "environment.json").read_text())
        for name, args in {"go": ["go", "version"], "python": ["python3", "--version"],
                           "allure": ["allure", "--version"],
                           "postgres": ["psql", os.environ["DATABASE_URL"], "-Atc", "SELECT version()"],
                           "migration": ["psql", os.environ["DATABASE_URL"], "-Atc", "SELECT max(version_id) FROM goose_db_version WHERE is_applied"]}.items():
            environment[name] = subprocess.check_output(args, text=True).strip()
        (OUTPUT / "environment.json").write_text(json.dumps(environment, indent=2))
        command(["go", "build", "-o", "/stand/marketplace-api", "./cmd/api"])
        shutil.copyfile("tests/config.yaml", "/stand/config.yaml")
    elif stage == "unit":
        (OUTPUT / "raw").mkdir(exist_ok=True)
        command([sys.executable, "-m", "pytest", "-q", "scripts/testing", "--junitxml", str(OUTPUT / "raw/python-unit.xml")])
        go_tests(stage, ["-count=1", "-shuffle=on", "./..."])
        if os.getenv("TEST_FAULT") == "unit":
            command(["python3", "-c", "import unittest; case = unittest.FunctionTestCase(lambda: unittest.TestCase().fail('injected unit failure')); result = unittest.TextTestRunner().run(unittest.TestSuite([case])); raise SystemExit(not result.wasSuccessful())"])
        command(["go", "vet", "./..."])
        command(["make", "test-unit-offline"])
        try:
            command(["make", "test-artifacts"])
        finally:
            if Path("artifacts/lab1").exists():
                shutil.copytree("artifacts/lab1", OUTPUT / "lab1", dirs_exist_ok=True)
    elif stage == "integration":
        go_tests(stage, ["-count=1", "-shuffle=on", "-p=1", "-tags=integration", "-run", "_Integration$",
                         "./internal/repository", "./internal/service"])
    elif stage == "e2e":
        command(["pytest", "-q", "tests/e2e", "--alluredir", "/run/allure-results",
                 "--junitxml", "/run/raw/e2e.xml"])
    elif stage == "cleanup":
        command(["python3", "-m", "tests.e2e.data", "cleanup-all"])
    elif stage == "guard-cleanup":
        from tests.e2e.data import check_cleanup_guards
        check_cleanup_guards()
    elif stage == "traffic":
        capture = subprocess.Popen(["tcpdump", "-Z", "root", "-i", "eth0", "-U", "-G", "10", "-W", "1",
                                    "-w", "/run/purchase.pcap", "tcp port 8080"],
                                   stdout=subprocess.DEVNULL, stderr=(OUTPUT / "capture.log").open("wb"))
        try:
            import time
            time.sleep(0.5)
            if capture.poll() is not None:
                raise RuntimeError("tcpdump could not start")
            command(["python3", "-m", "tests.e2e.purchase", "--curl"])
        finally:
            # Host AppArmor may deny signals to tcpdump. Native timed rotation
            # exits on the next packet; a read-only unauthenticated request supplies it.
            deadline = time.monotonic() + 20
            while capture.poll() is None and time.monotonic() < deadline:
                subprocess.run(["curl", "--silent", "--max-time", "1", "--output", "/dev/null",
                                "http://api:8080/api/v1/users/me"], check=False)
                time.sleep(0.2)
            if capture.poll() is None:
                raise RuntimeError("timed traffic capture did not finish; container cleanup will stop it")
        with (OUTPUT / "traffic-summary.tsv").open("wb") as output:
            command(["tshark", "-r", "/run/purchase.pcap", "-d", "tcp.port==8080,http", "-T", "fields",
                     "-e", "frame.number", "-e", "http.request.method", "-e", "http.request.uri",
                     "-e", "http.response.code", "-e", "http.request_in"], stdout=output)
        verify_capture(OUTPUT / "traffic-summary.tsv")
    elif stage == "report":
        results = OUTPUT / "allure-results"
        results.mkdir(exist_ok=True)
        import json
        import time
        environment = json.loads((OUTPUT / "environment.json").read_text())
        (results / "executor.json").write_text(json.dumps({"name": "Marketplace test runner", "type": "github" if os.getenv("CI") else "local",
            "buildOrder": int(time.time()), "buildName": environment["run"], "reportName": environment["run"]}))
        (results / "environment.properties").write_text("\n".join(f"{key}={value}" for key, value in environment.items()))
        for source in (OUTPUT / "raw").glob("*.xml"):
            if source.stem != "e2e":
                group = {"unit": "Unit", "integration": "Integration", "python-unit": "Infrastructure"}[source.stem]
                junit_to_allure(source, results, group)
                previous = results / source.name
                if previous.exists():
                    previous.unlink()  # Original JUnit stays in raw/; importing both formats duplicates cases.
        command(["allure", "generate", str(results), "--clean", "-o", "/run/allure-report"])
        public = OUTPUT / "public-allure-results"
        public_results(results, public)
        command(["allure", "generate", str(public), "--clean", "-o", "/run/public-allure-report"])
    else:
        raise ValueError(f"unknown stage {stage}")


if __name__ == "__main__":
    main(sys.argv[1])
