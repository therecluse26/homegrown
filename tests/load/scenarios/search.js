/**
 * Search scenario — exercises GET /v1/search?q=... (authenticated).
 *
 * Uses a shared session token from setup() to avoid redundant logins.
 * Rotates through representative search terms to avoid cache-only results.
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL } from '../helpers/config.js';
import { login, authParams } from '../helpers/auth.js';

export const options = {
  thresholds: {
    'http_req_duration{type:read}': ['p(95)<300'],
    'http_req_failed': ['rate<0.01'],
  },
};

const SEARCH_TERMS = [
  'charlotte mason',
  'classical',
  'phonics',
  'math curriculum',
  'nature journal',
  'history',
  'latin',
  'literature',
];

export function setup() {
  return { sessionToken: login() };
}

export default function (data) {
  const term = SEARCH_TERMS[Math.floor(Math.random() * SEARCH_TERMS.length)];

  const res = http.get(
    `${BASE_URL}/v1/search?q=${encodeURIComponent(term)}`,
    authParams(data.sessionToken, 'read'),
  );

  check(res, {
    'search status 200': (r) => r.status === 200,
  });

  sleep(1);
}
