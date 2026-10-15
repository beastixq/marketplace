from decimal import Decimal
from urllib.parse import parse_qs, urlsplit

import allure

from tests.e2e.data import PASSWORD
from tests.e2e.faults import after_checkout


@allure.title("Buyer purchases a product through delivery and review")
@allure.parent_suite("E2E")
def test_purchase_until_review(purchase_data):
    data = purchase_data
    api = data.api
    buyer_id = data.manifest["ids"]["buyer_id"]

    with allure.step("Buyer logs in and creates a fresh cart"):
        token = api.call("POST", "/auth/login", body={
            "email": data.manifest["emails"]["buyer"], "password": PASSWORD,
        })["token"]
        data.tokens["buyer"] = token
        api.call("POST", "/cart/items", 204, token=token, body={
            "product_id": data.product_id, "quantity": 2,
        })
        cart = api.call("GET", "/cart", token=token)
        assert cart["status"] == "draft"
        assert cart["user_id"] == buyer_id

    with allure.step("Checkout fixes price and creates an owned pending order"):
        orders = api.call("POST", "/orders", 201, token=token, body={
            "address_id": data.address_id,
        })["order_ids"]
        assert len(orders) == 1
        order_id = data.remember("order_id", orders[0])
        order = api.call("GET", f"/orders/{order_id}", token=token)
        assert order["id"] == order_id
        assert order["status"] == "pending"
        assert order["user_id"] == buyer_id
        assert order["seller_id"] == data.seller_id
        assert order["address_id"] == data.address_id
        assert Decimal(order["total_amount"]) == Decimal("251.00")
        items = api.call("GET", f"/orders/{order_id}/items", token=token)
        assert len(items) == 1
        assert items[0]["order_id"] == order_id
        assert items[0]["product_id"] == data.product_id
        assert items[0]["quantity"] == 2
        assert Decimal(items[0]["price_at_purchase"]) == Decimal("125.50")

    after_checkout()

    with allure.step("Mock bank payment is persisted as paid"):
        link = api.call("POST", f"/orders/{order_id}/payment-link", token=token)
        assert link["order_id"] == order_id
        payment_token = parse_qs(urlsplit(link["payment_url"]).query)["token"][0]
        api.call("POST", "/payments/callback/mock-bank", body={"token": payment_token})
        assert api.call("GET", f"/orders/{order_id}", token=token)["status"] == "paid"

    with allure.step("Seller ships the paid order"):
        api.call("POST", f"/orders/{order_id}/ship", 204, token=data.tokens["seller"])
        assert api.call("GET", f"/orders/{order_id}", token=token)["status"] == "shipped"

    with allure.step("Seller delivers the shipped order"):
        api.call("POST", f"/orders/{order_id}/deliver", 204, token=data.tokens["seller"])
        assert api.call("GET", f"/orders/{order_id}", token=token)["status"] == "delivered"

    with allure.step("Buyer posts and reads the delivered-purchase review"):
        review_id = data.remember("review_id", api.call("POST", "/reviews", 201, token=token, body={
            "product_id": data.product_id, "rating": 5, "comment": data.marker,
        })["id"])
        reviews = api.call("GET", f"/products/{data.product_id}/reviews?page=1&limit=20")
        matching = [review for review in reviews if review["id"] == review_id]
        assert len(matching) == 1
        assert matching[0]["user_id"] == buyer_id
        assert matching[0]["product_id"] == data.product_id
        assert matching[0]["rating"] == 5
        assert matching[0]["comment"] == data.marker
