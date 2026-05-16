/**
 * Baseline load test — 100 concurrent users for 5 minutes across all four key scenarios.
 *
 * Acceptance criteria [HOM-22]:
 *   - 100 VUs sustained for 5 minutes
 *   - p95 latency ≤ 300 ms for all read endpoints
 *   - Error rate < 1%
 *
 * Run:        k6 run tests/load/baseline.js
 * With env:   BASE_URL=https://staging.homegrown.academy k6 run tests/load/baseline.js
 * JSON output: k6 run --out json=results/baseline.json tests/load/baseline.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL } from './helpers/config.js';
import { login, fetchFirstStudentId, authParams } from './helpers/auth.js';

export const options = {
  // Four scenario types share 100 total VUs for 5 minutes.
  scenarios: {
    auth: {
      executor: 'constant-vus',
      vus: 10,
      duration: '5m',
      exec: 'authScenario',
      tags: { scenario: 'auth' },
    },
    marketplace: {
      executor: 'constant-vus',
      vus: 40,
      duration: '5m',
      exec: 'marketplaceScenario',
      tags: { scenario: 'marketplace' },
    },
    search: {
      executor: 'constant-vus',
      vus: 30,
      duration: '5m',
      exec: 'searchScenario',
      tags: { scenario: 'search' },
    },
    learning: {
      executor: 'constant-vus',
      vus: 20,
      duration: '5m',
      exec: 'learningScenario',
      tags: { scenario: 'learning' },
    },
  },
  thresholds: {
    // Global p95 target across all read-tagged requests.
    'http_req_duration{type:read}': ['p(95)<300'],
    // Per-scenario p95 targets.
    'http_req_duration{scenario:auth}': ['p(95)<500'],
    'http_req_duration{scenario:marketplace}': ['p(95)<300'],
    'http_req_duration{scenario:search}': ['p(95)<300'],
    'http_req_duration{scenario:learning}': ['p(95)<300'],
    // Overall error rate.
    http_req_failed: ['rate<0.01'],
  },
};

const LISTING_QUERIES = ['math', 'history', 'science', 'art', 'music', ''];
const SEARCH_TERMS = ['charlotte mason', 'classical', 'phonics', 'math curriculum', 'nature journal'];

// setup() runs once; all scenario functions receive the returned data object.
export function setup() {
  const sessionToken = login();
  const studentId = fetchFirstStudentId(sessionToken);
  return { sessionToken, studentId };
}

// ─── Scenario functions ───────────────────────────────────────────────────────

export function authScenario(data) {
  const res = http.get(`${BASE_URL}/v1/auth/me`, authParams(data.sessionToken, 'read'));
  check(res, { 'auth/me 200': (r) => r.status === 200 });
  sleep(1);
}

export function marketplaceScenario() {
  const query = LISTING_QUERIES[Math.floor(Math.random() * LISTING_QUERIES.length)];
  const url = query ? `${BASE_URL}/v1/listings?q=${encodeURIComponent(query)}` : `${BASE_URL}/v1/listings`;
  const res = http.get(url, { tags: { type: 'read' } });
  check(res, { 'listings 200': (r) => r.status === 200 });
  sleep(1);
}

export function searchScenario(data) {
  const term = SEARCH_TERMS[Math.floor(Math.random() * SEARCH_TERMS.length)];
  const res = http.get(`${BASE_URL}/v1/search?q=${encodeURIComponent(term)}`, authParams(data.sessionToken, 'read'));
  check(res, { 'search 200': (r) => r.status === 200 });
  sleep(1);
}

export function learningScenario(data) {
  if (!data.studentId) return;
  const res = http.get(
    `${BASE_URL}/v1/learning/students/${data.studentId}/activities`,
    authParams(data.sessionToken, 'read'),
  );
  check(res, { 'activities 200': (r) => r.status === 200 });
  sleep(1);
}
