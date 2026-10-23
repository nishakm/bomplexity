#!/usr/bin/env bash
# Part 2 demo, run in the SECOND terminal while `docker run -p 8080:8080 guestbook:dev` is up in the first.
. "$(dirname "$0")/demo-magic.sh"

TYPE_SPEED=30
DEMO_PROMPT="${GREEN}➜ ${CYAN}guestbook ${COLOR_RESET}"

clear
pe "curl -X POST localhost:8080/entries -H 'Content-Type: application/json' -d '{\"name\":\"nisha\",\"message\":\"hello BSidesPDX\"}'"
pe "curl localhost:8080/entries"
pe "curl localhost:8080/sbom"
pe "python3.14 ../utilities/spdx_tree.py ../goapp/sbom/image.spdx.json | less"
pe "python3.14 ../utilities/spdx_tree.py ../goapp/sbom/base-alpine_3.23.spdx.json | less"
