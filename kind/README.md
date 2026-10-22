# Demo cluster (kind)

Three commands run the whole demo on a local kind cluster: Postgres 18, the
app and nginx, reachable at <http://localhost:8080>. Run them from this folder.

Prerequisites: Docker, [kind](https://kind.sigs.k8s.io/), `kubectl`, `curl`,
the app image already built (`guestbook:dev`, see `../goapp`), and a `.env`
file in this folder (git-ignored):

```
cp .env.example .env      # then set a password; keep it the same in DATABASE_URL
```

```
make up       # cluster -> load image -> namespace -> secret -> Postgres -> app + nginx
make smoke    # POST an entry through nginx, read it back, fail loudly if either step fails
make down     # delete the whole cluster
```

- **From nothing:** `make up` takes about 1.5 minutes, most of it creating the
  cluster and pulling the Postgres and nginx images.
- **Safe to re-run:** every step is idempotent. A second `make up` takes a few
  seconds and changes nothing. `make down` works even if the cluster is gone.
- **Faster reset:** `make undeploy` deletes only the namespace (including the
  database volume), keeping the cluster; `make up` then redeploys in about 15
  seconds.
- **Individual steps:** `cluster`, `load`, `namespace`, `secret`, `deploy`, and
  `status` (pods, services, PVCs). `make help` (the default) lists everything.
- **The image is not built here.** `make load` stops with an error if
  `guestbook:dev` does not exist locally. Build it with `make image` in
  `../goapp`, or `docker build -t guestbook:dev ../goapp`.
- **Settings:** override on the command line, e.g. `make up CLUSTER_NAME=demo2`
  (`CLUSTER_NAME`, `NAMESPACE`, `IMAGE`, `KIND_CONFIG`, `MANIFESTS`,
  `ENV_FILE`, `SMOKE_URL`). `cluster.yaml` maps host port 8080, so only one
  cluster can use that port at a time.
- **Passwords:** the database password is fixed the first time Postgres starts.
  If you change it in `.env` later, run `make undeploy up` (or `make down up`)
  so the volume is recreated.
