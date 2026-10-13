import pytest

from tests.e2e.client import API
from tests.e2e.data import PurchaseData


@pytest.fixture
def purchase_data(request):
    data = PurchaseData(API())
    # Register before arrangement: partial fixture failures must still finalize.
    request.addfinalizer(data.cleanup)
    return data.prepare()
