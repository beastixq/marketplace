"""Test-result boundaries; Allure remains the report renderer and history owner."""

import hashlib
import json
import re
import time
import uuid
import xml.etree.ElementTree as ET
from pathlib import Path
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

SECRET_KEYS = {"password", "password_hash", "token", "access_token", "authorization", "jwt_secret"}


def redact(value):
    if isinstance(value, dict):
        return {key: "<redacted>" if key.lower() in SECRET_KEYS else redact(item)
                for key, item in value.items()}
    if isinstance(value, list):
        return [redact(item) for item in value]
    if isinstance(value, str):
        value = re.sub(r"Bearer\s+[^\s\"']+", "Bearer <redacted>", value, flags=re.I)
        if value.startswith(("http://", "https://")):
            url = urlsplit(value)
            query = [(key, "<redacted>" if key.lower() in SECRET_KEYS else item)
                     for key, item in parse_qsl(url.query, keep_blank_values=True)]
            value = urlunsplit((url.scheme, url.netloc, url.path, urlencode(query), url.fragment))
        return value
    return value


def leaf_junit(source, target):
    tree = ET.parse(source)
    for suite in tree.iter("testsuite"):
        cases = list(suite.findall("testcase"))
        for case in cases:
            name = case.get("name", "")
            children = [child for child in cases if child.get("name", "").startswith(name + "/")]
            if children:
                own_failure = case.find("failure") is not None or case.find("error") is not None
                child_failure = any(child.find("failure") is not None or child.find("error") is not None for child in children)
                if own_failure and not child_failure:
                    case.set("name", name + "/suite_cleanup")
                else:
                    suite.remove(case)
        leaves = suite.findall("testcase")
        suite.set("tests", str(len(leaves)))
        suite.set("failures", str(sum(case.find("failure") is not None for case in leaves)))
        suite.set("skipped", str(sum(case.find("skipped") is not None for case in leaves)))
    tree.write(target, encoding="utf-8", xml_declaration=True)


def junit_summary(path):
    summary = {"passed": 0, "failed": 0, "skipped": 0}
    for case in ET.parse(path).iter("testcase"):
        key = "passed"
        if case.find("skipped") is not None:
            key = "skipped"
        elif case.find("failure") is not None or case.find("error") is not None:
            key = "failed"
        summary[key] += 1
    return summary


def require_integration(path, packages):
    executed = set()
    skipped = []
    for line in Path(path).read_text().splitlines():
        event = json.loads(line)
        name = event.get("Test", "")
        if name.split("/")[0].endswith("_Integration"):
            if event.get("Action") == "pass":
                executed.add(event.get("Package"))
            elif event.get("Action") == "skip":
                skipped.append(name)
    missing = set(packages) - executed
    if missing or skipped:
        raise ValueError(f"integration did not execute fully: missing={sorted(missing)}, skipped={skipped}")


def stage_result(directory, stage, status, message=""):
    directory = Path(directory)
    directory.mkdir(parents=True, exist_ok=True)
    name = f"Test pipeline / {stage}"
    result = {"uuid": str(uuid.uuid4()), "name": name, "fullName": name,
              "historyId": hashlib.sha256(name.encode()).hexdigest(), "status": status,
              "stage": "finished", "start": int(time.time() * 1000), "stop": int(time.time() * 1000),
              "statusDetails": {"message": message},
              "labels": [{"name": "parentSuite", "value": "Infrastructure"}]}
    (directory / f'{result["uuid"]}-result.json').write_text(json.dumps(result))
    return result
