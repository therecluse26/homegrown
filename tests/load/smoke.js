/**
 * Smoke test — quick sanity check before full baseline runs.
 * 5 VUs, 30 seconds. Fails fast on any 5xx or threshold breach.
 *
 * Run: k6 run tests/load/smoke.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL } from './helpers/config.js';
import { login, fetchFirstStudentId, authParams } from './helpers/auth.js';

export const options = {
  vus: 5,
  duration: '30s',
  thresholds: {
    'http_req_duration{type:read}': ['p(95)<300'],
    'http_req_failed': ['rate<0.01'],
  },
};

export function setup() {
  const sessionToken = login();
  const studentId = fetchFirstStudentId(sessionToken);
  return { sessionToken, studentId };
}

export default function (data) {
  const params = authParams(data.sessionToken, 'read');

  // Health check (public)
  check(http.get(`${BASE_URL}/health`, { tags: { type: 'read' } }), {
    'health 200': (r) => r.status === 200,
  });

  // Auth/me
  check(http.get(`${BASE_URL}/v1/auth/me`, params), {
    'auth/me 200': (r) => r.status === 200,
  });

  // Public listings
  check(http.get(`${BASE_URL}/v1/listings`, { tags: { type: 'read' } }), {
    'listings 200': (r) => r.status === 200,
  });

  // Search
  check(http.get(`${BASE_URL}/v1/search?q=math`, params), {
    'search 200': (r) => r.status === 200,
  });

  // Learning activities (if a student exists)
  if (data.studentId) {
    check(
      http.get(`${BASE_URL}/v1/learning/students/${data.studentId}/activities`, params),
      { 'activities 200': (r) => r.status === 200 },
    );
  }

  sleep(1);
}
