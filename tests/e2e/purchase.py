import argparse
import os
import time
from decimal import Decimal
from pathlib import Path
from urllib.parse import parse_qs, urlsplit

import allure

from tests.e2e.client import API
from tests.e2e.data import PASSWORD, PurchaseData


def purchase(data):
    api = data.api
    with allure.step("Buyer logs in and creates a fresh cart"):
        token = api.call("POST", "/auth/login", body={"email": data.manifest["emails"]["buyer"], "password": PASSWORD})["token"]
        data.tokens["buyer"] = token
        api.call("POST", "/cart/items", 204, token=token, body={"product_id": data.product_id, "quantity": 2})
        cart = api.call("GET", "/cart", token=token)
        assert cart["status"] == "draft"
        assert cart["user_id"] == data.manifest["ids"]["buyer_id"]
    with allure.step("Checkout fixes price and creates an owned pending order"):
        orders = api.call("POST", "/orders", 201, token=token, body={"address_id": data.address_id})["order_ids"]
        assert len(orders) == 1
        order_id = data.remember("order_id", orders[0])
        order = api.call("GET", f"/orders/{order_id}", token=token)
        assert order["id"] == order_id and order["status"] == "pending"
        assert order["user_id"] == data.manifest["ids"]["buyer_id"]
        assert order["seller_id"] == data.seller_id and order["address_id"] == data.address_id
        assert Decimal(order["total_amount"]) == Decimal("251.00")
        items = api.call("GET", f"/orders/{order_id}/items", token=token)
        assert len(items) == 1 and items[0]["order_id"] == order_id
        assert items[0]["product_id"] == data.product_id and items[0]["quantity"] == 2
        assert Decimal(items[0]["price_at_purchase"]) == Decimal("125.50")
        gate = os.getenv("TEST_FAULT_GATE")
        if gate in ["e2e", "shared"]:
            Path(f"/run/{gate}-written").write_text("committed checkout")
            deadline = time.monotonic() + 180
            while gate == "e2e" or not Path("/run/shared-release").exists():
                if time.monotonic() >= deadline:
                    raise RuntimeError("acceptance gate was not released in 180 seconds")
                time.sleep(1)
    with allure.step("Mock bank payment is persisted as paid"):
        link = api.call("POST", f"/orders/{order_id}/payment-link", token=token)
        assert link["order_id"] == order_id
        payment_token = parse_qs(urlsplit(link["payment_url"]).query)["token"][0]
        api.call("POST", "/payments/callback/mock-bank", body={"token": payment_token})
        assert api.call("GET", f"/orders/{order_id}", token=token)["status"] == "paid"
    with allure.step("Seller ships and delivers the paid order"):
        for action, status in [("ship", "shipped"), ("deliver", "delivered")]:
            api.call("POST", f"/orders/{order_id}/{action}", 204, token=data.tokens["seller"])
            assert api.call("GET", f"/orders/{order_id}", token=token)["status"] == status
    with allure.step("Buyer posts and reads the delivered-purchase review"):
        review_id = data.remember("review_id", api.call("POST", "/reviews", 201, token=token,
            body={"product_id": data.product_id, "rating": 5, "comment": data.marker})["id"])
        reviews = api.call("GET", f"/products/{data.product_id}/reviews?page=1&limit=20")
        matching = [review for review in reviews if review["id"] == review_id]
        assert len(matching) == 1
        assert matching[0]["user_id"] == data.manifest["ids"]["buyer_id"]
        assert matching[0]["product_id"] == data.product_id and matching[0]["rating"] == 5
        assert matching[0]["comment"] == data.marker


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--curl", action="store_true")
    args = parser.parse_args()
    data = PurchaseData(API(use_curl=args.curl))
    try:
        data.prepare()
        purchase(data)
    finally:
        data.cleanup()
