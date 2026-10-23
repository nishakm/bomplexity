#!/usr/bin/env bash
# Part 1 demo, run in the SECOND terminal while `make run` (in goapp/) is up in the first.
. "$(dirname "$0")/demo-magic.sh"

TYPE_SPEED=30
DEMO_PROMPT="${GREEN}➜ ${CYAN}guestbook ${COLOR_RESET}"

clear
pe "curl -X POST localhost:18080/entries -H 'Content-Type: application/json' -d '{\"name\":\"nisha\",\"message\":\"hello BSidesPDX\"}'"
pe "curl localhost:18080/entries"
pe "curl localhost:18080/sbom"
pe "cd ../goapp/sbom"
pe "grype purl:runtime.txt"
