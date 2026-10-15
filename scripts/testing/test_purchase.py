import os
import tempfile
import threading
import unittest
from pathlib import Path
from unittest import mock

from tests.e2e import purchase


class PurchaseEntrypointTests(unittest.TestCase):
    def test_fixture_rejects_unknown_transport_before_creating_data(self):
        from tests.e2e.conftest import purchase_data

        with mock.patch.dict(os.environ, {"TEST_HTTP_TRANSPORT": "crul"}):
            with self.assertRaisesRegex(ValueError, "requests or curl"):
                purchase_data.__wrapped__(mock.Mock())

    def test_curl_uses_the_pytest_scenario_and_restores_transport(self):
        def run_pytest(arguments):
            self.assertEqual(os.environ["TEST_HTTP_TRANSPORT"], "curl")
            self.assertIn(str(Path(purchase.__file__).with_name("test_purchase.py")), arguments)
            return 1

        with mock.patch.dict(os.environ, {"TEST_HTTP_TRANSPORT": "requests"}):
            with mock.patch("pytest.main", side_effect=run_pytest):
                self.assertEqual(purchase.main(["--curl"]), 1)
            self.assertEqual(os.environ["TEST_HTTP_TRANSPORT"], "requests")

    def test_default_transport_and_environment_are_restored_on_pytest_error(self):
        with mock.patch.dict(os.environ, {}, clear=True):
            def run_pytest(arguments):
                self.assertEqual(os.environ["TEST_HTTP_TRANSPORT"], "requests")
                raise RuntimeError("pytest failed to initialize")

            with mock.patch("pytest.main", side_effect=run_pytest):
                with self.assertRaisesRegex(RuntimeError, "initialize"):
                    purchase.main([])
            self.assertNotIn("TEST_HTTP_TRANSPORT", os.environ)


class CheckoutFaultTests(unittest.TestCase):
    def test_normal_checkout_does_not_write_a_fault_marker(self):
        from tests.e2e.faults import after_checkout

        with tempfile.TemporaryDirectory() as directory:
            with mock.patch.dict(os.environ, {"TEST_FAULT_GATE": ""}):
                after_checkout(Path(directory))
            self.assertEqual(list(Path(directory).iterdir()), [])

    def test_shared_checkout_waits_for_release_after_publishing_committed_marker(self):
        from tests.e2e.faults import after_checkout

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            entered_wait = threading.Event()
            release_wait = threading.Event()
            errors = []

            def pause(seconds):
                entered_wait.set()
                if not release_wait.wait(5):
                    raise RuntimeError("test did not release checkout")

            def run_gate():
                try:
                    after_checkout(root)
                except BaseException as error:
                    errors.append(error)

            with mock.patch.dict(os.environ, {"TEST_FAULT_GATE": "shared"}):
                with mock.patch("tests.e2e.faults.time.sleep", side_effect=pause):
                    worker = threading.Thread(target=run_gate)
                    worker.start()
                    try:
                        self.assertTrue(entered_wait.wait(5))
                        self.assertEqual((root / "shared-written").read_text(), "committed checkout")
                        self.assertTrue(worker.is_alive())
                        (root / "shared-release").touch()
                    finally:
                        release_wait.set()
                        worker.join(5)
            self.assertFalse(worker.is_alive())
            self.assertEqual(errors, [])

    def test_interruption_gate_times_out_even_if_shared_release_exists(self):
        from tests.e2e.faults import after_checkout

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "shared-release").touch()
            with mock.patch.dict(os.environ, {"TEST_FAULT_GATE": "e2e"}):
                with mock.patch("tests.e2e.faults.time.monotonic", side_effect=[0, 181]):
                    with self.assertRaisesRegex(RuntimeError, "180 seconds"):
                        after_checkout(root)
            self.assertEqual((root / "e2e-written").read_text(), "committed checkout")


if __name__ == "__main__":
    unittest.main()
