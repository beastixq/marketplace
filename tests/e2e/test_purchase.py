import allure

from tests.e2e.purchase import purchase


@allure.title("Buyer purchases a product through delivery and review")
@allure.parent_suite("E2E")
def test_purchase_until_review(purchase_data):
    purchase(purchase_data)
