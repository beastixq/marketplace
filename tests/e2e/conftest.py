import os

import pytest

from tests.e2e.client import API
from tests.e2e.data import PurchaseData


@pytest.fixture
def purchase_data(request):
    transport = os.getenv("TEST_HTTP_TRANSPORT", "requests")
    if transport not in {"requests", "curl"}:
        raise ValueError("TEST_HTTP_TRANSPORT must be requests or curl")
    data = PurchaseData(API(use_curl=transport == "curl"))
    # Register before arrangement: partial fixture failures must still finalize.
    request.addfinalizer(data.cleanup)
    return data.prepare()
