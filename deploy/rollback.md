# Rollback Procedure

Every production deploy is tagged `sha-<git-sha>` in GitHub Container Registry
(`ghcr.io/<owner>/homegrown-academy`). Rolling back means re-deploying
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
IMAGE=ghcr.io/<owner>/homegrown-academy   # match your GitHub org/user

docker pull "${IMAGE}:sha-${GOOD_SHA}"
IMAGE="${IMAGE}" IMAGE_TAG="sha-${GOOD_SHA}" docker compose -f docker-compose.prod.yml up -d --no-deps --wait app
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

Goose migrations run **at startup** via the Go server binary (`goose.Up()` at boot).
Migrations are append-only; rolling back the app image does **not** roll back the schema.

- If the new schema is backward-compatible with the old binary, the image rollback
  above is sufficient.
- If the schema is **not** backward-compatible, run the down migration manually
  before re-deploying the old image. Goose is not installed as a standalone binary
  in the runtime image — use the official goose container against the live database:

```bash
cd /opt/homegrown-academy

# Load DATABASE_URL from the env file
. .env.production

# Run one step down using the pressly/goose container
docker run --rm \
  --network host \
  -e GOOSE_DBSTRING="${DATABASE_URL}" \
  ghcr.io/pressly/goose:latest \
  -dir /migrations postgres down-to <target-version>
```

Then redeploy the old image as in Step 2.

---

## Required GitHub Secrets

Secrets are scoped per GitHub Environment (`staging`, `production`).
Add them at: **GitHub repo → Settings → Environments → {env} → Environment secrets**.

### SSH / infra (both environments)

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

### App secrets (per environment, prefix `STAGING_` or `PRODUCTION_`)

| Secret suffix | Example value | Required |
|---------------|---------------|----------|
| `DATABASE_URL` | `postgres://homegrown:pass@postgres:5432/homegrown?sslmode=disable` | ✅ |
| `REDIS_URL` | `redis://redis:6379` | ✅ |
| `POSTGRES_USER` | `homegrown` | ✅ |
| `POSTGRES_PASSWORD` | random 32-char string | ✅ |
| `POSTGRES_DB` | `homegrown` | ✅ |
| `AUTH_ADMIN_URL` | `http://kratos:4434` | ✅ |
| `AUTH_PUBLIC_URL` | `http://kratos:4433` | ✅ |
| `AUTH_WEBHOOK_SECRET` | random 32-char string (not the dev default) | ✅ |
| `CORS_ALLOWED_ORIGINS` | `https://app.homegrown.example` | ✅ |
| `UNSUBSCRIBE_SECRET` | random 32-char string (not the dev default) | ✅ |
| `OBJECT_STORAGE_PUBLIC_URL` | `https://media.homegrown.example` | ✅ prod |
| `HYPERSWITCH_BASE_URL` | `https://sandbox.hyperswitch.io` | billing |
| `HYPERSWITCH_API_KEY` | from Hyperswitch dashboard | billing |
| `HYPERSWITCH_WEBHOOK_KEY` | from Hyperswitch dashboard | billing |
| `HYPERSWITCH_BILLING_PROFILE_ID` | from Hyperswitch dashboard | billing |
| `HYPERSWITCH_MONTHLY_PRICE_ID` | from Hyperswitch dashboard | billing |
| `HYPERSWITCH_ANNUAL_PRICE_ID` | from Hyperswitch dashboard | billing |
| `BILLING_WEBHOOK_SECRET` | random 32-char string | billing |
| `COPPA_CHARGE_CENTS` | `50` ($0.50 default — override if needed) | optional |
| `POSTMARK_SERVER_TOKEN` | from Postmark dashboard | email |
| `OBJECT_STORAGE_ENDPOINT` | `https://<account>.r2.cloudflarestorage.com` | media |
| `OBJECT_STORAGE_REGION` | `auto` | media |
| `OBJECT_STORAGE_BUCKET` | bucket name | media |
| `OBJECT_STORAGE_ACCESS_KEY_ID` | R2/S3 access key | media |
| `OBJECT_STORAGE_SECRET_ACCESS_KEY` | R2/S3 secret key | media |
| `ERROR_REPORTING_DSN` | Sentry DSN (omit to disable) | optional |

### Required GitHub Environments

| Environment | Protection rule |
|-------------|----------------|
| `staging` | None — auto-deploys on every push to `main` |
| `production` | **Required reviewers** — at least one human must approve before deploy proceeds |

Configure at: **GitHub repo → Settings → Environments → production → Deployment protection rules → Required reviewers**.

---

## Server bootstrap (first-time only)

Before the first CI deploy can succeed, the target server needs:

```bash
# Create the deploy directory
sudo mkdir -p /opt/homegrown-academy
sudo chown <deploy-user>: /opt/homegrown-academy

# Copy docker-compose.prod.yml to the server
scp docker-compose.prod.yml <user>@<host>:/opt/homegrown-academy/

# The deploy pipeline writes .env.production automatically on every deploy.
# No manual secret file needed after the first run.
```

Ory Kratos is not included in `docker-compose.prod.yml` — run it as a separate
compose stack or managed service and point `AUTH_ADMIN_URL` / `AUTH_PUBLIC_URL` at it.
