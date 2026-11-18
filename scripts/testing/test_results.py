import json
import tempfile
import unittest
from pathlib import Path

from scripts.testing import results


class ResultTests(unittest.TestCase):
    def test_ci_report_keeps_failed_unit_and_skipped_downstream_jobs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "inputs/testing-unit"
            (source / "raw").mkdir(parents=True)
            (source / "raw/unit.xml").write_text('<testsuites><testsuite><testcase name="p1"><failure/></testcase></testsuite></testsuites>')
            (source / "stages.json").write_text(json.dumps({"unit": "failed"}))
            stale = root / "inputs/testing-integration/allure-results"
            stale.mkdir(parents=True)
            (stale / "old-result.json").write_text(json.dumps({"name": "old", "status": "passed"}))
            output = root / "report"
            jobs = {"unit": {"result": "failure"}, "integration": {"result": "skipped"}, "e2e": {"result": "skipped"}}
            actual = results.collect_ci_results(root / "inputs", output, jobs, {"run": "r1"})
            self.assertEqual(actual, {"unit": "failed", "integration": "skipped", "e2e": "skipped"})
            self.assertTrue((output / "raw/unit.xml").exists())
            markers = [json.loads(path.read_text()) for path in (output / "allure-results").glob("*-result.json")]
            self.assertEqual({case["name"]: case["status"] for case in markers}, {
                "Test pipeline / unit": "failed", "Test pipeline / integration": "skipped", "Test pipeline / e2e": "skipped"})

    def test_ci_report_does_not_hide_recovery_failure_behind_passing_tests(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "inputs/testing-integration"
            source.mkdir(parents=True)
            (source / "stages.json").write_text(json.dumps({"integration": "passed"}))
            (source / "recovery.json").write_text(json.dumps({"restored": False}))
            jobs = {"unit": {"result": "skipped"}, "integration": {"result": "failure"}, "e2e": {"result": "skipped"}}
            actual = results.collect_ci_results(root / "inputs", root / "report", jobs, {"run": "r1"})
            self.assertEqual(actual["integration"], "broken")
            self.assertEqual(actual["e2e"], "skipped")

    def test_ci_report_rejects_success_without_stage_results(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            jobs = {"unit": {"result": "success"}, "integration": {"result": "skipped"}, "e2e": {"result": "skipped"}}
            actual = results.collect_ci_results(root / "inputs", root / "report", jobs, {"run": "r1"})
            self.assertEqual(actual["unit"], "broken")

    def test_ci_report_preserves_interruption_and_original_e2e_steps(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "inputs/testing-e2e"
            (source / "allure-results").mkdir(parents=True)
            case = {"uuid": "r1", "name": "p1", "fullName": "p1", "status": "broken",
                    "steps": [{"name": "checkout", "status": "passed"}]}
            (source / "allure-results/r1-result.json").write_text(json.dumps(case))
            (source / "stages.json").write_text(json.dumps({"e2e": "broken"}))
            jobs = {"unit": {"result": "skipped"}, "integration": {"result": "skipped"}, "e2e": {"result": "failure"}}
            output = root / "report"
            actual = results.collect_ci_results(root / "inputs", output, jobs, {"run": "r1"})
            self.assertEqual(actual["e2e"], "broken")
            self.assertEqual(json.loads((output / "allure-results/r1-result.json").read_text()), case)

    def test_go_parent_suites_are_not_counted_twice(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "input.xml"
            target = Path(directory) / "output.xml"
            source.write_text('<testsuites><testsuite name="pkg"><testcase name="TestCart"/>'
                              '<testcase name="TestCart/one"/><testcase name="TestCart/two">'
                              '<failure>insufficient stock</failure></testcase></testsuite></testsuites>')
            results.leaf_junit(source, target)
            self.assertEqual(results.junit_summary(target), {"passed": 1, "failed": 1, "skipped": 0})
            self.assertEqual(results.junit_summary(source), {"passed": 1, "failed": 1, "skipped": 0})

    def test_allure_import_separates_stages_without_changing_junit_history(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "integration.xml"
            source.write_text('<testsuites><testsuite name="pkg" timestamp="2026-10-15T12:00:00Z">'
                              '<testcase classname="pkg" name="TestCart_Integration"/>'
                              '<testcase classname="pkg" name="TestCart_Integration/one" time="0.125"/>'
                              '<testcase classname="pkg" name="TestCart_Integration/two"><error message="database lost">trace</error></testcase>'
                              '</testsuite></testsuites>')
            target = root / "results"
            target.mkdir()
            results.junit_to_allure(source, target, "Integration")
            cases = [json.loads(path.read_text()) for path in target.glob("*-result.json")]
            self.assertEqual(len(cases), 2)
            first = next(case for case in cases if case["name"].endswith("/one"))
            labels = {label["name"]: label["value"] for label in first["labels"]}
            self.assertEqual(labels["parentSuite"], "Integration")
            self.assertEqual(labels["suite"], "pkg")
            self.assertEqual(first["historyId"], "pkg:pkg#TestCart_Integration/one")
            self.assertEqual(first["stop"] - first["start"], 125)
            self.assertEqual({case["status"] for case in cases}, {"passed", "broken"})
            results.junit_to_allure(source, target, "Integration")
            self.assertEqual(len(list(target.glob("*-result.json"))), 2, "report regeneration must not duplicate cases")

    def test_empty_or_skipped_integration_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "events.jsonl"
            for events in [[], [{"Action": "skip", "Test": "TestCart_Checkout_Integration", "Package": "pkg"}]]:
                path.write_text("\n".join(json.dumps(event) for event in events))
                with self.assertRaisesRegex(ValueError, "integration"):
                    results.require_integration(path, ["pkg"])

    def test_suite_cleanup_failure_is_not_hidden_by_passing_children(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "input.xml"
            target = Path(directory) / "output.xml"
            source.write_text('<testsuites><testsuite name="pkg"><testcase name="TestCart">'
                              '<failure>cleanup failed</failure></testcase>'
                              '<testcase name="TestCart/buy"/></testsuite></testsuites>')
            results.leaf_junit(source, target)
            self.assertEqual(results.junit_summary(target), {"passed": 1, "failed": 1, "skipped": 0})

    def test_both_integration_packages_must_execute(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "events.jsonl"
            path.write_text(json.dumps({"Action": "pass", "Test": "TestCart_Checkout_Integration", "Package": "service"}))
            with self.assertRaisesRegex(ValueError, "repository"):
                results.require_integration(path, ["service", "repository"])

    def test_secrets_are_removed_without_hiding_business_values(self):
        value = {"password": "private", "headers": {"Authorization": "Bearer private"},
                 "url": "http://api/mock?token=private&order_id=9", "amount": "249.00"}
        cleaned = results.redact(value)
        self.assertNotIn("private", json.dumps(cleaned))
        self.assertEqual(cleaned["amount"], "249.00")
        self.assertIn("order_id=9", cleaned["url"])

    def test_public_report_excludes_attachments_traces_environment_and_history_secrets(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, target = root / "raw", root / "public"
            source.mkdir()
            (source / "one-result.json").write_text(json.dumps({
                "uuid": "one", "historyId": "stable", "name": "purchase", "status": "failed",
                "labels": [{"name": "parentSuite", "value": "E2E"}, {"name": "host", "value": "private-host"}],
                "statusDetails": {"trace": "private-password"},
                "attachments": [{"source": "private-http.json"}],
                "steps": [{"name": "Checkout", "status": "passed", "attachments": [{"source": "private-http.json"}]}],
                "parameters": [{"name": "token", "value": "private-token"}],
            }))
            (source / "private-http.json").write_text("private-password")
            history = source / "history"
            history.mkdir()
            (history / "history.json").write_text(json.dumps({"stable": {
                "statistic": {"failed": 1}, "items": [{"uid": "old", "status": "failed", "time": {},
                    "statusDetails": "private-password", "reportUrl": "http://private-host"}]}}))
            results.public_results(source, target)
            serialized = "".join(path.read_text() for path in target.rglob("*.json"))
            self.assertNotIn("private-", serialized)
            case = json.loads((target / "one-result.json").read_text())
            self.assertEqual(case["status"], "failed")
            self.assertEqual(case["historyId"], "stable")
            self.assertEqual(case["steps"], [{"name": "Checkout", "status": "passed"}])
            self.assertEqual(case["labels"], [{"name": "parentSuite", "value": "E2E"}])
            self.assertFalse((target / "private-http.json").exists())

    def test_stage_marker_has_stable_history_and_real_status(self):
        with tempfile.TemporaryDirectory() as directory:
            first = results.stage_result(Path(directory), "integration", "broken", "killed")
            second = results.stage_result(Path(directory), "integration", "skipped", "unit failed")
            self.assertEqual(first["historyId"], second["historyId"])
            self.assertNotEqual(first["uuid"], second["uuid"])
            self.assertEqual(first["status"], "broken")
            self.assertEqual(second["status"], "skipped")


if __name__ == "__main__":
    unittest.main()
