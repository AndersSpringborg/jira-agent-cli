#!/usr/bin/env python3
"""Trim the Jira Cloud OpenAPI spec to only the paths we need, resolving all referenced schemas."""

import json
import re
import sys

NEEDED_PATHS = [
    "/rest/api/3/issue",
    "/rest/api/3/issue/{issueIdOrKey}",
    "/rest/api/3/issue/{issueIdOrKey}/assignee",
    "/rest/api/3/issue/{issueIdOrKey}/comment",
    "/rest/api/3/issue/{issueIdOrKey}/transitions",
    "/rest/api/3/issue/{issueIdOrKey}/watchers",
    "/rest/api/3/issue/{issueIdOrKey}/worklog",
    "/rest/api/3/issue/{issueIdOrKey}/remotelink",
    "/rest/api/3/issueLink",
    "/rest/api/3/issueLink/{linkId}",
    "/rest/api/3/issueLinkType",
    "/rest/api/3/search/jql",
    "/rest/api/3/field",
    "/rest/api/3/project",
    "/rest/api/3/project/{projectIdOrKey}",
    "/rest/api/3/project/{projectIdOrKey}/versions",
    "/rest/api/3/myself",
    "/rest/api/3/user/search",
    "/rest/api/3/user/assignable/search",
    "/rest/api/3/user",
    "/rest/api/3/serverInfo",
    "/rest/api/3/issue/createmeta",
    "/rest/api/3/issue/createmeta/{projectIdOrKey}/issuetypes",
    "/rest/api/3/issue/createmeta/{projectIdOrKey}/issuetypes/{issueTypeId}",
]


def find_refs(obj, refs=None):
    """Recursively find all $ref values in a JSON structure."""
    if refs is None:
        refs = set()
    if isinstance(obj, dict):
        if "$ref" in obj:
            refs.add(obj["$ref"])
        for v in obj.values():
            find_refs(v, refs)
    elif isinstance(obj, list):
        for item in obj:
            find_refs(item, refs)
    return refs


def resolve_ref(spec, ref):
    """Get the object at a $ref path like '#/components/schemas/Foo'."""
    parts = ref.lstrip("#/").split("/")
    obj = spec
    for p in parts:
        obj = obj[p]
    return obj


def collect_schemas(spec, paths_subset):
    """Collect all schemas transitively referenced from the given paths."""
    needed_schemas = set()
    to_process = set()

    # Find initial refs from paths
    refs = find_refs(paths_subset)
    for ref in refs:
        m = re.match(r"#/components/schemas/(.+)", ref)
        if m:
            to_process.add(m.group(1))

    # Transitively resolve
    while to_process:
        name = to_process.pop()
        if name in needed_schemas:
            continue
        needed_schemas.add(name)
        if name in spec.get("components", {}).get("schemas", {}):
            schema = spec["components"]["schemas"][name]
            refs = find_refs(schema)
            for ref in refs:
                m = re.match(r"#/components/schemas/(.+)", ref)
                if m and m.group(1) not in needed_schemas:
                    to_process.add(m.group(1))

    return needed_schemas


def collect_security_schemes(spec, paths_subset):
    """Collect security schemes referenced from paths."""
    schemes = set()
    for path_obj in paths_subset.values():
        for op in path_obj.values():
            if isinstance(op, dict) and "security" in op:
                for sec in op["security"]:
                    schemes.update(sec.keys())
    # Also check top-level security
    if "security" in spec:
        for sec in spec["security"]:
            schemes.update(sec.keys())
    return schemes


def main():
    with open("openapi/openapi-jira-cloud.json") as f:
        spec = json.load(f)

    # Filter paths
    trimmed_paths = {}
    for p in NEEDED_PATHS:
        if p in spec["paths"]:
            trimmed_paths[p] = spec["paths"][p]
        else:
            print(f"WARNING: path {p} not found in spec", file=sys.stderr)

    # Collect schemas
    needed_schemas = collect_schemas(spec, trimmed_paths)
    print(
        f"Found {len(trimmed_paths)} paths, {len(needed_schemas)} schemas",
        file=sys.stderr,
    )

    # Collect security schemes
    security_schemes = collect_security_schemes(spec, trimmed_paths)

    # Build trimmed spec
    trimmed = {
        "openapi": spec["openapi"],
        "info": spec["info"],
        "servers": spec.get("servers", [{"url": "https://your-domain.atlassian.net"}]),
        "paths": trimmed_paths,
        "components": {
            "schemas": {
                name: spec["components"]["schemas"][name]
                for name in sorted(needed_schemas)
                if name in spec["components"]["schemas"]
            }
        },
    }

    # Add security schemes if any
    if security_schemes:
        sec_schemes = {}
        for name in security_schemes:
            if name in spec.get("components", {}).get("securitySchemes", {}):
                sec_schemes[name] = spec["components"]["securitySchemes"][name]
        if sec_schemes:
            trimmed["components"]["securitySchemes"] = sec_schemes

    # Keep top-level security
    if "security" in spec:
        trimmed["security"] = spec["security"]

    with open("openapi/openapi-jira-cloud-trimmed.json", "w") as f:
        json.dump(trimmed, f, indent=2)

    print(f"Written to openapi/openapi-jira-cloud-trimmed.json", file=sys.stderr)


if __name__ == "__main__":
    main()
