#!/usr/bin/env python3
"""Print the packages of an SPDX 2.3 JSON SBOM and their relationships as an ASCII tree.

usage: spdx_tree.py SBOM.json
"""
import json
import sys
from collections import defaultdict

# relationship type -> True if the edge points parent -> child as written,
# False if it must be flipped ("A DEPENDENCY_OF B" means B depends on A).
EDGES = {"DEPENDS_ON": True, "CONTAINS": True, "DESCRIBES": True,
         "DEPENDENCY_OF": False, "CONTAINED_BY": False, "DESCRIBED_BY": False}

doc = json.load(open(sys.argv[1]))
name = {p["SPDXID"]: f'{p["name"]} - {p.get("versionInfo", "unknown")}' for p in doc["packages"]}

children = defaultdict(set)
has_parent = set()
for r in doc.get("relationships", []):
    if r["relationshipType"] not in EDGES:
        continue
    a, b = r["spdxElementId"], r["relatedSpdxElement"]
    if not EDGES[r["relationshipType"]]:
        a, b = b, a
    if a in name and b in name:  # ignore files and the document itself
        children[a].add(b)
        has_parent.add(b)

def walk(node, prefix, path):
    kids = sorted(children[node], key=lambda k: name[k])
    for i, k in enumerate(kids):
        last = i == len(kids) - 1
        print(prefix + ("`-- " if last else "+-- ") + name[k] + (" (cycle)" if k in path else ""))
        if k not in path:
            walk(k, prefix + ("    " if last else "|   "), path | {k})

for root in sorted((n for n in name if n not in has_parent), key=lambda n: name[n]):
    print(name[root])
    walk(root, "", {root})
