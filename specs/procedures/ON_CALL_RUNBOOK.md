# Homegrown Academy — On-Call Runbook

## Who Gets Paged

| Severity | First Responder | Escalation | Contact |
|----------|----------------|------------|---------|
| **Critical** | On-call engineer | CTO (after 15 min no ack) | PagerDuty rotation |
| **Warning** | On-call engineer | — | Slack `#alerts` channel |

Current on-call rotation: managed in PagerDuty under the **Homegrown Academy** service.
PagerDuty service URL: configure during PagerDuty account setup.

---

## Escalation Path

```
Alert fires
  └─▶ PagerDuty pages on-call engineer (phone + SMS)
        └─▶ Ack within 15 min → investigate and resolve
        └─▶ No ack after 15 min → escalate to CTO
              └─▶ No ack after 30 min → escalate to all engineers (broadcast)
```

---

## Alert Playbooks

### High Error Rate

**Alert:** `HighErrorRate` — HTTP 5xx rate > 1% over 5 min.

**Checklist:**
1. Check Grafana → HTTP Overview dashboard for which routes are erroring.
2. Check application logs: `docker compose logs -f --tail=200 api`
3. Check the `/health` endpoint: `curl -s http://HOST:3500/health | jq .`
4. Check database connectivity: `psql -h localhost -p 5932 -U homegrown -c '\l'`
5. Check Redis: `redis-cli -p 6879 ping`
6. If a recent deploy triggered it: roll back with `git revert HEAD && make deploy`.
7. If a migration caused it: check `migrations/` for the latest file and revert manually.

**Common causes:**
- Missing environment variable after a deploy — check `.env` on the host.
- Database migration ran without matching code deploy (or vice versa).
- Redis flushed or restarted — session cache miss; usually self-heals.
- Kratos unavailable — auth endpoints return 503.

---

### High Latency

**Alert:** `HighP95Latency` — p95 latency > 1s over 5 min.

**Checklist:**
1. Open Grafana → HTTP Overview → sort by p95 to identify the slow route(s).
2. Check for slow queries:
   ```sql
   SELECT pid, now() - query_start AS duration, query
   FROM pg_stat_activity
   WHERE state = 'active'
   ORDER BY duration DESC
   LIMIT 10;
   ```
3. Check Redis latency: `redis-cli -p 6879 --latency-history -i 1`
4. If latency is isolated to media/upload routes, check Cloudflare R2 connectivity.
5. If load is unusually high, check for bot traffic via `/v1/admin/health`.

**Common causes:**
- Missing or invalid database index on a high-traffic query.
- N+1 queries introduced in a new feature.
- Background job batch running during peak hours.
- Cold-start after restart — caches need to warm; usually resolves in < 5 min.

---

### Health Check Failing

**Alert:** `HealthCheckFailing` — `GET /health` returns non-200.

**Checklist:**
1. Check which dependency is unhealthy:
   ```bash
   curl -s http://HOST:3500/health | jq .checks
   ```
   - `"database": "error: ..."` → PostgreSQL is down or unreachable.
   - `"redis": "error: ..."` → Redis is down or unreachable.
2. Check Docker containers: `docker compose ps` — look for `Exit` or `unhealthy` state.
3. Restart the unhealthy service: `docker compose restart postgres` or `docker compose restart redis`.
4. If the API container itself is down: `docker compose logs api --tail=100`.
5. If all dependencies are healthy but `/health` still returns 503, restart the API process.

---

### API Down

**Alert:** `APIDown` — Prometheus cannot scrape `/metrics`.

**Checklist:**
1. Check if the process is running: `systemctl status homegrown-api` or `docker compose ps`.
2. Check crash logs: `journalctl -u homegrown-api -n 100 --no-pager` or `docker compose logs api --tail=100`.
3. Check for OOM kills: `dmesg | grep -i oom | tail -20`.
4. Restart the service: `systemctl restart homegrown-api` or `docker compose restart api`.
5. If the restart fails, check for port conflicts: `ss -tlnp | grep 3500`.
6. Notify users via status page if downtime exceeds 5 minutes.

---

## Common Incident Playbooks

### Backup Failure

**Symptoms:** `homegrown-backup.service` failed; no new backup in R2/S3 within the expected window.

```bash
# Check last backup status
journalctl -u homegrown-backup.service -n 50

# Run backup manually to diagnose
sudo systemctl start homegrown-backup.service
journalctl -fu homegrown-backup.service

# Verify backup credentials are set
cat /opt/homegrown-academy/.env.backup   # contains RCLONE_* or S3 vars
```

### Database Connection Pool Exhaustion

**Symptoms:** High error rate; logs show `pgx: conn pool exhausted` or `too many clients`.

```bash
# Check current connections
psql -h localhost -p 5932 -U homegrown -c \
  "SELECT count(*), state FROM pg_stat_activity GROUP BY state;"

# Kill idle connections older than 10 minutes
psql -h localhost -p 5932 -U homegrown -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
   WHERE state = 'idle' AND query_start < now() - interval '10 minutes';"
```

Long-term: increase `DB_MAX_OPEN_CONNS` env var or reduce background job parallelism.

### Runaway Background Job

**Symptoms:** p95 latency spike, DB CPU high, asynq queue growing.

```bash
# Inspect asynq queues
asynq stats -uri redis://localhost:6879

# Pause a queue
asynq queue pause default --uri redis://localhost:6879

# Purge stuck tasks (only if safe to discard)
asynq task purge --queue default --uri redis://localhost:6879
```

### Full Disk on Host

**Symptoms:** API returns 500; logs show `no space left on device`.

```bash
du -sh /var/lib/docker/volumes/*   # find large volumes
docker system prune -f             # remove dangling images/layers
```

If Prometheus TSDB is growing unbounded, `--storage.tsdb.retention.time=7d` in the
Prometheus service command caps it to 7 days.

### Kratos Auth Outage

**Symptoms:** All login/registration requests fail with 503.

```bash
# Check Kratos health
curl http://localhost:4934/health/ready

# Restart Kratos
docker compose restart kratos

# If migration is needed after an upgrade
docker compose run --rm kratos migrate sql -e --yes --config /etc/kratos/kratos.yml
```

---

## Post-Incident Process

1. **Within 1 hour:** Update the incident ticket with timeline and immediate fix.
2. **Within 24 hours:** Write a brief post-mortem (5 whys, timeline, action items).
3. **Within 1 week:** Implement preventive measures from action items; add a regression test if applicable.
4. **Update this runbook** if a new failure mode was discovered.

---

## Key Metrics Dashboards

| Dashboard | URL |
|-----------|-----|
| API latency & error rate | Grafana → **Homegrown API** dashboard |
| Background jobs (asynq) | Grafana → **Asynq Workers** dashboard |
| Postgres | Grafana → **PostgreSQL** dashboard |
| Uptime | UptimeRobot / Better Uptime status page |

---

## Contact List

| Role | Contact | Availability |
|------|---------|--------------|
| Engineering Lead | _fill in_ | PagerDuty + phone |
| Primary On-Call | _rotates weekly_ | PagerDuty |
| Secondary On-Call | _rotates weekly_ | PagerDuty |
| Database Admin | _fill in_ | Slack `#incidents` |

---

## Useful Links (local dev defaults)

| Resource | URL |
|----------|-----|
| Prometheus | `http://localhost:9090` |
| Grafana | `http://localhost:3000` (admin / homegrown) |
| Alertmanager | `http://localhost:9093` |
| Blackbox Exporter | `http://localhost:9115` |
| Health endpoint | `http://localhost:3500/health` |
| Admin health | `http://localhost:3500/v1/admin/health` |

## PagerDuty / Alertmanager Setup (production)

1. Create a PagerDuty service and copy the **Events API v2 integration key**.
2. Set `PAGERDUTY_INTEGRATION_KEY=<key>` in the production environment.
3. Set `ALERTMANAGER_WEBHOOK_URL=<slack-webhook-url>` for the fallback Slack notification.
4. Alertmanager reads these at startup from `monitoring/alertmanager.yml` — the values are
   injected via environment variable substitution; no secrets are committed to the repo.
