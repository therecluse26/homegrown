/**
 * Auth scenario — validates the login flow and /v1/auth/me endpoint.
 *
 * Each VU logs in fresh (simulates real concurrent login pressure) and then
 * hits /v1/auth/me to verify the session. This exercises both the Kratos
 * login path and the backend's session validation middleware.
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL } from '../helpers/config.js';
import { login } from '../helpers/auth.js';

export const options = {
  thresholds: {
    'http_req_duration{type:read}': ['p(95)<300'],
    'http_req_failed': ['rate<0.01'],
  },
};

export default function () {
  // Login and immediately check /auth/me — each VU performs a full auth cycle.
  const sessionToken = login();

  const meRes = http.get(`${BASE_URL}/v1/auth/me`, {
    headers: { 'X-Session-Token': sessionToken },
    tags: { type: 'read' },
  });

  check(meRes, {
    'auth/me status 200': (r) => r.status === 200,
    'auth/me has familyId': (r) => r.json('familyId') !== undefined || r.json('family_id') !== undefined,
  });

  sleep(1);
}
