"""Acceptance-only checkpoint after checkout assertions and manifest persistence."""

import os
import time
from pathlib import Path


def after_checkout(directory=Path("/run")):
    gate = os.getenv("TEST_FAULT_GATE")
    if gate not in {"e2e", "shared"}:
        return

    (directory / f"{gate}-written").write_text("committed checkout")
    deadline = time.monotonic() + 180
    while gate == "e2e" or not (directory / "shared-release").exists():
        if time.monotonic() >= deadline:
            raise RuntimeError("acceptance gate was not released in 180 seconds")
        time.sleep(1)
