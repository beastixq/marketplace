"""Test-result boundaries; Allure remains the report renderer and history owner."""

import hashlib
import json
import re
import time
import uuid
import xml.etree.ElementTree as ET
from datetime import datetime, timezone
from decimal import Decimal
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


def leaf_cases(source):
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
    return tree


def leaf_junit(source, target):
    leaf_cases(source).write(target, encoding="utf-8", xml_declaration=True)


def junit_to_allure(source, target, stage):
    """Add hierarchy without renaming JUnit identities or inventing test steps."""
    for suite in leaf_cases(source).iter("testsuite"):
        stamp = suite.get("timestamp")
        started = datetime.fromisoformat(stamp.replace("Z", "+00:00")) if stamp else datetime.now(timezone.utc)
        if started.tzinfo is None:
            started = started.replace(tzinfo=timezone.utc)
        start = int(started.timestamp() * 1000)
        for case in suite.findall("testcase"):
            name, package = case.get("name", "Unknown test case"), case.get("classname", suite.get("name", ""))
            # Allure 2's JUnit plugin uses this exact identity; labels must not reset history.
            history = f'{suite.get("name")}:{package}#{name}'
            status, details = "passed", {}
            for tag, outcome in [("failure", "failed"), ("error", "broken"), ("skipped", "skipped")]:
                element = case.find(tag)
                if element is not None:
                    status = outcome
                    details = {"message": element.get("message", ""), "trace": element.text or ""}
                    break
            if case.get("status") == "notrun":
                status = "skipped"
            result = {"uuid": str(uuid.uuid4()), "name": name, "fullName": f"{package}#{name}",
                      "historyId": history, "status": status, "stage": "finished", "statusDetails": details,
                      "start": start, "stop": start + int(Decimal(case.get("time", "0")) * 1000),
                      "labels": [{"name": "parentSuite", "value": stage},
                                 {"name": "suite", "value": suite.get("name", package)},
                                 {"name": "package", "value": package}, {"name": "testClass", "value": package}]}
            filename = hashlib.sha256((stage + history).encode()).hexdigest() + "-result.json"
            (Path(target) / filename).write_text(json.dumps(result))


def junit_summary(path):
    summary = {"passed": 0, "failed": 0, "skipped": 0}
    for case in leaf_cases(path).iter("testcase"):
        key = "passed"
        if case.find("skipped") is not None:
            key = "skipped"
        elif case.find("failure") is not None or case.find("error") is not None:
            key = "failed"
        summary[key] += 1
    return summary


def public_results(source, target):
    """Publish an allowlisted view, never the diagnostic result directory itself."""
    source, target = Path(source), Path(target)
    target.mkdir(exist_ok=True)
    def step(value):
        safe = {key: value[key] for key in ["name", "status", "start", "stop"] if key in value}
        if value.get("steps"):
            safe["steps"] = [step(child) for child in value["steps"]]
        return safe
    labels = {"parentSuite", "suite", "subSuite", "package", "testClass", "framework", "language"}
    for path in source.glob("*-result.json"):
        value = json.loads(path.read_text())
        safe = {key: value[key] for key in ["uuid", "name", "fullName", "historyId", "status", "stage", "start", "stop"] if key in value}
        safe["labels"] = [label for label in value.get("labels", []) if label["name"] in labels]
        if value.get("steps"):
            safe["steps"] = [step(child) for child in value["steps"]]
        (target / path.name).write_text(json.dumps(safe))
    executor = source / "executor.json"
    if executor.exists():
        value = json.loads(executor.read_text())
        safe = {key: value[key] for key in ["name", "type", "buildOrder", "buildName", "reportName"] if key in value}
        (target / "executor.json").write_text(json.dumps(safe))
    if (source / "history").exists():
        history = target / "history"
        history.mkdir(exist_ok=True)
        path = source / "history/history.json"
        if path.exists():
            safe = {}
            for identity, value in json.loads(path.read_text()).items():
                safe[identity] = {"statistic": value["statistic"], "items": [
                    {key: item[key] for key in ["uid", "status", "time"] if key in item}
                    for item in value.get("items", [])]}
            (history / "history.json").write_text(json.dumps(safe))
        for name in ["history-trend.json", "categories-trend.json", "duration-trend.json", "retry-trend.json"]:
            path = source / "history" / name
            if path.exists():
                (history / name).write_bytes(path.read_bytes())


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
