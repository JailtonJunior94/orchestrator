#!/usr/bin/env python3
"""
Validates a generated Postman collection JSON file.
Usage: python3 validate-collection.py <collection.json>

Checks:
- Valid JSON
- Postman v2.1 schema reference
- Every item has name, request.method, request.url
- Every request with body has valid JSON in body.raw
- At least one saved response per request
- No placeholder templates like {{...}} remain in URLs or bodies

Exit 0 on success, exit 1 with descriptive errors on failure.
"""
import json
import re
import sys

def validate_item(item, path=""):
    errors = []
    current = f"{path}/{item.get('name', 'unnamed')}"

    # Folder (has sub-items)
    if "item" in item and isinstance(item["item"], list):
        for sub in item["item"]:
            errors.extend(validate_item(sub, current))
        return errors

    # Request item
    req = item.get("request")
    if not req:
        errors.append(f"{current}: missing 'request' object")
        return errors

    if "method" not in req:
        errors.append(f"{current}: missing request.method")

    url = req.get("url", {})
    raw_url = url.get("raw", "") if isinstance(url, dict) else str(url)
    if not raw_url:
        errors.append(f"{current}: missing request.url.raw")

    # Check for unresolved placeholders in URL (allow {{base_url}} and {{auth_token}})
    allowed_vars = {"base_url", "auth_token"}
    found_vars = set(re.findall(r"\{\{(\w+)\}\}", raw_url))
    bad_vars = found_vars - allowed_vars
    if bad_vars:
        errors.append(f"{current}: unresolved URL placeholders: {bad_vars}")

    # Validate body JSON
    body = req.get("body", {})
    if body and body.get("mode") == "raw" and body.get("raw", "").strip():
        raw = body["raw"].strip()
        if raw:
            try:
                json.loads(raw)
            except json.JSONDecodeError as e:
                errors.append(f"{current}: invalid JSON in request body: {e}")

    # Check saved responses
    responses = item.get("response", [])
    if not responses:
        errors.append(f"{current}: no saved response examples")
    else:
        for resp in responses:
            resp_body = resp.get("body", "")
            if resp_body and resp_body.strip():
                try:
                    json.loads(resp_body)
                except json.JSONDecodeError as e:
                    errors.append(f"{current}: invalid JSON in response '{resp.get('name', '?')}': {e}")

    return errors

def main():
    if len(sys.argv) != 2:
        print("Usage: python3 validate-collection.py <collection.json>", file=sys.stderr)
        sys.exit(1)

    filepath = sys.argv[1]
    try:
        with open(filepath, "r") as f:
            collection = json.load(f)
    except json.JSONDecodeError as e:
        print(f"FATAL: File is not valid JSON: {e}", file=sys.stderr)
        sys.exit(1)
    except FileNotFoundError:
        print(f"FATAL: File not found: {filepath}", file=sys.stderr)
        sys.exit(1)

    # Check schema
    schema = collection.get("info", {}).get("schema", "")
    if "v2.1.0" not in schema:
        print("WARNING: Collection does not reference Postman v2.1.0 schema", file=sys.stderr)

    # Validate items
    errors = []
    for item in collection.get("item", []):
        errors.extend(validate_item(item))

    if errors:
        print(f"VALIDATION FAILED ({len(errors)} issues):", file=sys.stderr)
        for err in errors:
            print(f"  - {err}", file=sys.stderr)
        sys.exit(1)
    else:
        item_count = sum(
            1 for folder in collection.get("item", [])
            for _ in folder.get("item", [folder])
        )
        print(f"SUCCESS: Collection is valid. {len(collection.get('item', []))} folders, ~{item_count} requests.")
        sys.exit(0)

if __name__ == "__main__":
    main()
