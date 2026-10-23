#!/usr/bin/env bash
# Part 3 demo, run from demo/ after running `make up` from kind/ in a different terminal.
. "$(dirname "$0")/demo-magic.sh"

TYPE_SPEED=30
DEMO_PROMPT="${GREEN}➜ ${CYAN}guestbook ${COLOR_RESET}"

clear
pe "curl -X POST localhost:8080/entries -H 'Content-Type: application/json' -d '{\"name\":\"nisha\",\"message\":\"hello BSidesPDX\"}'"
pe "curl -X POST localhost:8080/entries -H 'Content-Type: application/json' -d '{\"name\":\"audience\",\"message\":\"Hi Nisha!\"}'"
pe "curl localhost:8080/entries"
pe "curl localhost:8080/sbom"
pe "python3.14 ../utilities/spdx_tree.py ../kind/sbom/nginx_1.28.spdx.json | less"
pe "python3.14 ../utilities/spdx_tree.py ../kind/sbom/postgres_18.spdx.json | less"
