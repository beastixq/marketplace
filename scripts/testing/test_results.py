import json
import tempfile
import unittest
from pathlib import Path

from scripts.testing import results


class ResultTests(unittest.TestCase):
    def test_go_parent_suites_are_not_counted_twice(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "input.xml"
            target = Path(directory) / "output.xml"
            source.write_text('<testsuites><testsuite name="pkg"><testcase name="TestCart"/>'
                              '<testcase name="TestCart/one"/><testcase name="TestCart/two">'
                              '<failure>insufficient stock</failure></testcase></testsuite></testsuites>')
            results.leaf_junit(source, target)
            self.assertEqual(results.junit_summary(target), {"passed": 1, "failed": 1, "skipped": 0})

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
