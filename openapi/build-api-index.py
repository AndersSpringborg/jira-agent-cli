#!/usr/bin/env python3
"""Build compact API endpoint indexes for `jira api --list`.

Reads the local full Jira Cloud platform spec and downloads the official
Jira Server/Data Center platform spec and the Jira Software (agile) spec,
then emits compact indexes (method, path, summary) embedded into the binary
via internal/apispec.

Usage: python3 openapi/build-api-index.py
"""

import gzip
import json
import os
import urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
OUT_DIR = os.path.join(HERE, "..", "internal", "apispec")

CLOUD_PLATFORM_LOCAL = os.path.join(HERE, "openapi-jira-cloud.json")
# Jira Software Data Center spec covers both the platform v2 API and the
# agile 1.0 API; its paths are relative to the /rest context.
SERVER_DC_URL = "https://dac-static.atlassian.com/server/jira/platform/jira_software_dc_10007_swagger.v3.json"
CLOUD_AGILE_URL = "https://dac-static.atlassian.com/cloud/jira/software/swagger.v3.json"

HTTP_METHODS = {"get", "post", "put", "patch", "delete"}


def load_url(url):
    req = urllib.request.Request(
        url, headers={"User-Agent": "jira-cli-index-builder", "Accept": "application/json"}
    )
    with urllib.request.urlopen(req) as resp:
        data = resp.read()
    if data[:2] == b"\x1f\x8b":
        data = gzip.decompress(data)
    return json.loads(data)


def load_file(path):
    with open(path) as f:
        return json.load(f)


def index_spec(spec, path_prefix=""):
    """Extract [{method, path, summary}] from an OpenAPI spec."""
    entries = []
    for path, ops in spec.get("paths", {}).items():
        for method, op in ops.items():
            if method not in HTTP_METHODS or not isinstance(op, dict):
                continue
            summary = (op.get("summary") or "").strip()
            entries.append(
                {
                    "method": method.upper(),
                    "path": path_prefix + path,
                    "summary": summary,
                }
            )
    entries.sort(key=lambda e: (e["path"], e["method"]))
    return entries


def write_index(name, entries):
    out = os.path.join(OUT_DIR, name)
    with open(out, "w") as f:
        json.dump(entries, f, separators=(",", ":"), ensure_ascii=False)
        f.write("\n")
    print(f"wrote {out}: {len(entries)} endpoints")


def main():
    cloud_platform = index_spec(load_file(CLOUD_PLATFORM_LOCAL))
    cloud_agile = index_spec(load_url(CLOUD_AGILE_URL))
    server = index_spec(load_url(SERVER_DC_URL), path_prefix="/rest")

    write_index("index_cloud.json", cloud_platform + cloud_agile)
    write_index("index_server.json", server)


if __name__ == "__main__":
    main()
