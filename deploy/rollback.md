# Rollback Procedure

Every production deploy is tagged `sha-<git-sha>` in GitHub Container Registry
(`ghcr.io/homegrown-academy/homegrown-academy`). Rolling back means re-deploying
any previously pushed tag — no rebuild required.

---

## 1. Identify the last good commit

```bash
# On your local machine — find the commit before the bad deploy
git log --oneline main | head -20
```

Note the 40-character SHA of the last known-good commit.  
You can also find the tag in the GitHub Actions deploy run that was green:
**Actions → Deploy → Build & Push → extract the `sha-tag` output**.

---

## 2. Pull and redeploy that image

SSH to the affected server and run:

```bash
cd /opt/homegrown-academy

GOOD_SHA=<40-char-sha>        # replace with actual SHA
IMAGE=ghcr.io/homegrown-academy/homegrown-academy

docker pull "${IMAGE}:sha-${GOOD_SHA}"
IMAGE_TAG="sha-${GOOD_SHA}" docker compose -f docker-compose.prod.yml up -d --no-deps --wait app
```

`--wait` blocks until the container's health check passes (up to 40 s start period + 3
retries × 30 s = ~130 s). If it exits non-zero the rollback failed; check logs:

```bash
docker compose -f docker-compose.prod.yml logs app --tail 100
```

---

## 3. Verify

```bash
curl -sf https://<your-domain>/health | jq .
```

Expected: HTTP 200 with `{"status":"healthy","checks":{...}}`.

---

## 4. Block the bad commit (optional but recommended)

Once stable, open a GitHub branch protection rule or revert the bad commit so CI
cannot re-deploy it automatically:

```bash
# Revert on main (preferred — keeps history clean)
git revert <bad-sha> --no-edit
git push origin main
```

The revert commit triggers a new deploy through the normal pipeline, so staging
is validated first.

---

## Database rollbacks

Goose migrations run **at startup** (`goose up`) and are append-only.  
Rolling back the app image does **not** roll back the schema.

- If the new schema is backward-compatible with the old binary, the image rollback
  above is sufficient.
- If the schema is not backward-compatible, apply the goose down migration manually
  before re-deploying the old image:

```bash
# Exec into a temporary container with the old image to run goose
docker run --rm --env-file .env.production \
  --network homegrown-academy_default \
  ghcr.io/homegrown-academy/homegrown-academy:sha-${GOOD_SHA} \
  /bin/sh -c 'goose -dir /app/migrations postgres "$DATABASE_URL" down'
```

Then redeploy the old image as in Step 2.

---

## Required GitHub Secrets (for the deploy workflow)

| Secret | Description |
|--------|-------------|
| `STAGING_SSH_HOST` | Staging server hostname or IP |
| `STAGING_SSH_USER` | SSH username on staging server |
| `STAGING_SSH_KEY` | ED25519 private key for staging SSH |
| `STAGING_URL` | Base URL of staging app (e.g. `https://staging.homegrown.example`) |
| `PRODUCTION_SSH_HOST` | Production server hostname or IP |
| `PRODUCTION_SSH_USER` | SSH username on production server |
| `PRODUCTION_SSH_KEY` | ED25519 private key for production SSH |
| `PRODUCTION_URL` | Base URL of production app (e.g. `https://homegrown.example`) |

## Required GitHub Environments

| Environment | Protection rule |
|-------------|----------------|
| `staging` | None — auto-deploys on every push to `main` |
| `production` | **Required reviewers** — at least one human must approve before deploy proceeds |

Configure at: **GitHub repo → Settings → Environments → production → Deployment protection rules → Required reviewers**.
