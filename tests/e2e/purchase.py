"""Run the pytest purchase scenario with the curl transport for packet capture."""

import argparse
import os
from pathlib import Path


def main(argv=None):
    import pytest

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--curl", action="store_true")
    args = parser.parse_args(argv)
    previous = os.environ.get("TEST_HTTP_TRANSPORT")
    os.environ["TEST_HTTP_TRANSPORT"] = "curl" if args.curl else "requests"
    try:
        return pytest.main(["-q", str(Path(__file__).with_name("test_purchase.py"))])
    finally:
        if previous is None:
            os.environ.pop("TEST_HTTP_TRANSPORT", None)
        else:
            os.environ["TEST_HTTP_TRANSPORT"] = previous


if __name__ == "__main__":
    raise SystemExit(main())
