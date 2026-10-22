#!/usr/bin/env python3
"""Print the packages of SPDX 2.3 JSON SBOMs and their relationships as an ASCII tree.

usage: spdx_tree.py [SBOM.json]      (reads stdin if no file, or "-", is given)

The input may hold several SPDX documents one after another (for example one
per line); each is printed under a "== <document name>" heading.
"""
import json
import sys
from collections import defaultdict

# relationship type -> True if the edge points parent -> child as written,
# False if it must be flipped ("A DEPENDENCY_OF B" means B depends on A).
EDGES = {"DEPENDS_ON": True, "CONTAINS": True, "DESCRIBES": True,
         "DEPENDENCY_OF": False, "CONTAINED_BY": False, "DESCRIBED_BY": False}


def show(doc):
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


path = sys.argv[1] if len(sys.argv) > 1 else "-"
text = sys.stdin.read() if path == "-" else open(path).read()

docs, decoder, i = [], json.JSONDecoder(), 0
while True:
    while i < len(text) and text[i].isspace():
        i += 1
    if i >= len(text):
        break
    doc, i = decoder.raw_decode(text, i)
    docs.append(doc)

for n, doc in enumerate(docs):
    if len(docs) > 1:
        print(("\n" if n else "") + f'== {doc.get("name", "sbom")}')
    show(doc)
