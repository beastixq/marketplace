"""API arrangement and fail-closed PostgreSQL cleanup for one synthetic case."""

import json
import base64
import os
import subprocess
import socket
import sys
import uuid
from pathlib import Path

from tests.e2e.client import API

PASSWORD = "Synthetic-password-42"


def save_manifest(path, manifest):
    temporary = Path(path).with_suffix(".tmp")
    temporary.write_text(json.dumps(manifest, indent=2))
    temporary.replace(path)


class PurchaseData:
    def __init__(self, api):
        self.api = api
        self.marker = "e2e-" + os.environ["RUN_ID"] + "-" + uuid.uuid4().hex[:8]
        self.path = Path(os.getenv("TEST_OUTPUT", "/run")) / "fixtures" / (self.marker + ".json")
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self.manifest = {"run_id": os.environ["RUN_ID"], "marker": self.marker, "emails": {
            role: f"{self.marker}-{role}@example.test" for role in ["buyer", "seller"]},
            "ids": {}, "cleaned": False}
        self.tokens = {}
        self.save()

    def save(self):
        save_manifest(self.path, self.manifest)

    def remember(self, name, value):
        self.manifest["ids"][name] = value
        self.save()
        return value

    def prepare(self):
        for role in ["buyer", "seller"]:
            result = self.api.call("POST", "/auth/register", body={"email": self.manifest["emails"][role],
                "password": PASSWORD, "full_name": "Test " + self.marker, "role": role})
            self.tokens[role] = result["token"]
            credential_path = self.path.with_suffix(".credentials")
            credential_path.touch(mode=0o600, exist_ok=True)
            credential_path.write_text(json.dumps(self.tokens))
            user = self.api.call("GET", "/users/me", token=self.tokens[role])
            assert user["email"] == self.manifest["emails"][role] and user["role"] == role
            # Current /users/me omits id; the server-issued JWT exposes UserID.
            payload = self.tokens[role].split(".")[1]
            user_id = json.loads(base64.urlsafe_b64decode(payload + "=" * (-len(payload) % 4)))["UserID"]
            self.remember(role + "_id", user_id)
            if os.getenv("TEST_FAULT") == "fixture" and role == "buyer":
                raise RuntimeError("requested partial API-fixture failure after persisted buyer")
        self.seller_id = self.remember("seller_profile_id", self.api.call("POST", "/sellers", 201,
            token=self.tokens["seller"], body={"company_name": self.marker})["id"])
        self.product_id = self.remember("product_id", self.api.call("POST", "/products", 201,
            token=self.tokens["seller"], body={"seller_id": self.seller_id, "name": self.marker,
                                             "price": "125.50", "stock_quantity": 3})["id"])
        self.address_id = self.remember("address_id", self.api.call("POST", "/addresses", 201,
            token=self.tokens["buyer"], body={"city": "Test city", "street": self.marker,
                                            "house": "1", "zip_code": "123456"})["id"])
        return self

    def cleanup(self):
        cleanup_manifest(self.path, self.api)


def cleanup_manifest(path, api=None):
    manifest = json.loads(Path(path).read_text())
    if manifest["cleaned"]:
        verify_deleted_tokens(path, api)
        return
    if not manifest.get("database_cleaned"):
        sql = Path(__file__).with_name("cleanup.sql").read_text()
        result = subprocess.run(["psql", os.environ["DATABASE_URL"], "--no-psqlrc", "--set", "ON_ERROR_STOP=1",
                                 "--set", "manifest=" + json.dumps(manifest)], input=sql, text=True, capture_output=True)
        diagnostics = Path(path).with_suffix(".cleanup.log")
        diagnostics.write_text(result.stdout + result.stderr)
        if result.returncode:
            raise RuntimeError(f"owned-data cleanup refused/failed; see {diagnostics.name}")
        manifest["cache_product_ids"] = [int(line.split("=", 1)[1]) for line in result.stdout.splitlines()
                                         if line.startswith("CACHE_PRODUCT=")]
        manifest["database_cleaned"] = True
        # Preserve confirmed keys so a transient Redis failure can be retried after SQL deletion.
        save_manifest(path, manifest)
    # Keys are limited to confirmed product IDs returned by the guarded SQL, not arbitrary manifest input.
    for product_id in manifest["cache_product_ids"]:
        values = [b"DEL", f"products:{product_id}".encode()]
        request = b"*2\r\n" + b"".join(b"$" + str(len(value)).encode() + b"\r\n" + value + b"\r\n" for value in values)
        with socket.create_connection(("cache", 6379), timeout=5) as connection:
            connection.sendall(request)
            if not connection.recv(100).startswith(b":"):
                raise RuntimeError("owned product-cache cleanup failed")
    manifest["cleaned"] = True
    save_manifest(path, manifest)
    verify_deleted_tokens(path, api)


def verify_deleted_tokens(path, api=None):
    credentials = Path(path).with_suffix(".credentials")
    if credentials.exists():
        for token in json.loads(credentials.read_text()).values():
            (api or API()).call("GET", "/users/me", 401, token=token)
        credentials.unlink()


def check_cleanup_guards():
    own, other = PurchaseData(API()), PurchaseData(API())
    foreign_item = None
    try:
        own.prepare()
        other.prepare()
        other.api.call("POST", "/cart/items", 204, token=other.tokens["buyer"],
                       body={"product_id": own.product_id, "quantity": 1})
        cart = other.api.call("GET", "/cart", token=other.tokens["buyer"])
        items = other.api.call("GET", f'/orders/{cart["id"]}/items', token=other.tokens["buyer"])
        foreign_item = items[0]["id"]
        try:
            own.cleanup()
        except RuntimeError:
            diagnostics = own.path.with_suffix(".cleanup.log").read_text()
            assert "foreign references to run data" in diagnostics
        else:
            raise AssertionError("cleanup deleted data with foreign references")
        # Neither side may have lost data when the cleanup was refused.
        own.api.call("GET", "/users/me", token=own.tokens["buyer"])
        other.api.call("GET", f'/orders/{cart["id"]}/items', token=other.tokens["buyer"])
        assert own.api.call("GET", f"/products/{own.product_id}")["name"] == own.marker
        other.api.call("DELETE", f"/cart/items/{foreign_item}", 204, token=other.tokens["buyer"])
        foreign_item = None
    finally:
        if foreign_item is not None:
            other.api.call("DELETE", f"/cart/items/{foreign_item}", 204, token=other.tokens["buyer"])
        own.cleanup()
        other.cleanup()


if __name__ == "__main__":
    if sys.argv[1] != "cleanup-all":
        raise ValueError("expected cleanup-all")
    for manifest_path in (Path(os.getenv("TEST_OUTPUT", "/run")) / "fixtures").glob("*.json"):
        cleanup_manifest(manifest_path)
