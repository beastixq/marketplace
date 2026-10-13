import json
import os
import subprocess
from pathlib import Path

import allure
import requests

from scripts.testing.results import redact


class API:
    def __init__(self, base_url=None, use_curl=False):
        self.base_url = (base_url or os.environ["BASE_URL"]).rstrip("/")
        self.use_curl = use_curl
        self.session = requests.Session()
        self.session.trust_env = False

    def call(self, method, path, status=200, token=None, body=None):
        url = self.base_url + "/api/v1" + path
        headers = {"Content-Type": "application/json"}
        if token:
            headers["Authorization"] = "Bearer " + token
        with allure.step(f"{method} {path}"):
            if self.use_curl:
                config = ["url = " + json.dumps(url), "request = " + json.dumps(method)]
                config.extend("header = " + json.dumps(f"{key}: {value}") for key, value in headers.items())
                if body is not None:
                    config.append("data = " + json.dumps(json.dumps(body)))
                process = subprocess.run(["curl", "--silent", "--show-error", "--max-time", "15",
                                          "--config", "-", "--write-out", "\n%{http_code}"],
                                         input="\n".join(config), text=True, capture_output=True, check=True)
                raw, code = process.stdout.rsplit("\n", 1)
                code = int(code)
            else:
                response = self.session.request(method, url, headers=headers, json=body, timeout=15)
                raw, code = response.text, response.status_code
            try:
                content = json.loads(raw) if raw else None
            except json.JSONDecodeError:
                content = "<non-JSON response>"
            exchange = redact({"method": method, "url": url, "headers": headers, "request": body,
                               "status": code, "response": content})
            serialized = json.dumps(exchange, ensure_ascii=False)
            allure.attach(serialized, "HTTP exchange (redacted)", allure.attachment_type.JSON)
            directory = Path(os.getenv("TEST_OUTPUT", "/run"))
            directory.mkdir(parents=True, exist_ok=True)
            with (directory / "http-exchanges.jsonl").open("a") as output:
                output.write(serialized + "\n")
            assert code == status, f"{method} {path}: expected HTTP {status}, got {code}; {exchange['response']}"
            return content
