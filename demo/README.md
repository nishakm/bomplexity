# SBOM complexity demo

## Part 1 - An SBOM for a Go App

1.`cd goapp` and `make run`
2. run `./part1.sh`

## Part 2 - An SBOM for a containerized Go App

1. run `docker run -p 8080:8080 guestbook:dev`
2. run `./part2.sh`
3. interrupt docker

## Part 3 - An SBOM for deployed Kubernetes cluster

1. run `cd kind` and `make up`
2. run `./part3.sh`
