#!/usr/bin/env python3
"""Fetch the registry-published SPDX SBOM of every image running in a namespace.

Lists the pods' image digests with kubectl, then asks the registry for each
image's SBOM with `docker buildx imagetools inspect`. Prints one SPDX JSON
document per line on stdout, so it can be piped into spdx_tree.py:

    k8s_image_sbom.py -n sbom-demo | spdx_tree.py

Images without a published SBOM (for example ones loaded into kind from the
local Docker daemon) are reported on stderr and skipped.

With --local DIR no registry (and no docker) is needed: the SBOMs are read from
saved files in DIR instead. For each running image it looks for, in order:
  1. <name>@sha256_<digest>.spdx.json   exact match for the running image
  2. <name>_<tag>.spdx.json             matches the tag only (warns: the tag may
                                        have moved since the file was saved)
where <name> drops docker.io/library/ and turns "/" and ":" into "_", e.g.
nginx_1.28.spdx.json or nginx@sha256_146adea4....spdx.json.
Note: images put into kind with `kind load` report a local "import-<date>@sha256:..."
imageID instead of their registry digest, so they can only be matched by tag
(and the registry lookup cannot work for them).

usage: k8s_image_sbom.py [-n NAMESPACE] [--context CONTEXT] [--local DIR]
"""
import argparse
import json
import os
import subprocess
import sys


def run(*cmd):
    return subprocess.run(cmd, capture_output=True, text=True, timeout=120)


ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
ap.add_argument("-n", "--namespace", default="sbom-demo")
ap.add_argument("--context", help="kubectl context (default: the current one)")
ap.add_argument("--local", metavar="DIR", help="read saved SBOM files from DIR instead of asking the registry")
args = ap.parse_args()
kubectl = ["kubectl"] + (["--context", args.context] if args.context else [])

pods = run(*kubectl, "-n", args.namespace, "get", "pods", "-o", "json")
nodes = run(*kubectl, "get", "nodes", "-o", "json")
if pods.returncode or nodes.returncode:
    sys.exit("kubectl failed: " + (pods.stderr or nodes.stderr).strip())

# node name -> "os/arch", the key imagetools uses for per-platform SBOMs
platform = {n["metadata"]["name"]: "{operatingSystem}/{architecture}".format(**n["status"]["nodeInfo"])
            for n in json.loads(nodes.stdout)["items"]}

images = {}  # digest reference -> (image name, platform)
for pod in json.loads(pods.stdout)["items"]:
    status = pod["status"]
    for c in status.get("initContainerStatuses", []) + status.get("containerStatuses", []):
        ref = c["imageID"].removeprefix("docker-pullable://")
        images[ref] = (c["image"], platform[pod["spec"]["nodeName"]])

def safe(ref):
    """Image reference -> file name stem: docker.io/library/nginx:1.28 -> nginx_1.28"""
    ref = ref.removeprefix("docker.io/").removeprefix("library/")
    return ref.replace("@sha256:", "@sha256_").replace("/", "_").replace(":", "_")


def from_registry(name, ref, plat):
    r = run("docker", "buildx", "imagetools", "inspect", ref, "--format", "{{json .SBOM}}")
    if r.returncode:
        print(f"  skipped: registry lookup failed ({r.stderr.strip().splitlines()[-1]})", file=sys.stderr)
        return None
    doc = ((json.loads(r.stdout) or {}).get(plat) or {}).get("SPDX")
    if not doc:
        print(f"  skipped: no SPDX SBOM published for {plat}", file=sys.stderr)
    return doc


def from_directory(name, ref, plat):
    by_digest = os.path.join(args.local, safe(ref) + ".spdx.json")
    by_tag = os.path.join(args.local, safe(name) + ".spdx.json")
    for path, exact in ((by_digest, True), (by_tag, False)):
        if os.path.exists(path):
            if not exact:
                print(f"  using {path}: matched by tag only, cannot confirm it is the running digest", file=sys.stderr)
            doc = json.load(open(path))
            return doc.get("predicate", doc)  # accept a raw BuildKit attestation too
    print(f"  skipped: no {os.path.basename(by_digest)} or {os.path.basename(by_tag)} in {args.local}", file=sys.stderr)
    return None


found = 0
for ref, (name, plat) in sorted(images.items(), key=lambda kv: kv[1][0]):
    print(f"{name}: looking up SBOM for {ref}", file=sys.stderr)
    doc = (from_directory if args.local else from_registry)(name, ref, plat)
    if not doc:
        continue
    doc["name"] = name  # the registry's own name is just "sbom"
    print(json.dumps(doc, separators=(",", ":")))
    found += 1

if not found:
    sys.exit("no SBOMs found")
