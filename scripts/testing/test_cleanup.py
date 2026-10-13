import importlib.util
import json
import os
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import MagicMock, patch

from scripts.testing.controller import ROOT


class CleanupTests(unittest.TestCase):
    def test_cache_cleanup_can_retry_after_postgres_rows_have_been_deleted(self):
        # API calls are not this unit's boundary; load the fixture without pytest dependencies.
        with patch.dict("sys.modules", {"tests.e2e.client": SimpleNamespace(API=MagicMock())}):
            spec = importlib.util.spec_from_file_location("cleanup_fixture", ROOT / "tests/e2e/data.py")
            fixture = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(fixture)
        with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, DATABASE_URL="test-dsn"):
            path = Path(directory) / "fixture.json"
            path.write_text(json.dumps({"cleaned": False}))
            sql = SimpleNamespace(returncode=0, stdout="CACHE_PRODUCT=9\n", stderr="")
            connection = MagicMock()
            connection.__enter__.return_value.recv.return_value = b":1\r\n"
            with patch.object(fixture.subprocess, "run", return_value=sql) as database, \
                    patch.object(fixture.socket, "create_connection", side_effect=[OSError("Redis unavailable"), connection]):
                with self.assertRaises(OSError):
                    fixture.cleanup_manifest(path)
                fixture.cleanup_manifest(path)
                self.assertEqual(database.call_count, 1)
                connection.__enter__.return_value.sendall.assert_called_once()
            self.assertTrue(json.loads(path.read_text())["cleaned"])
