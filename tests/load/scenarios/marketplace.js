/**
 * Marketplace scenario — exercises the public GET /v1/listings browse endpoint.
 *
 * Listings are publicly accessible (pub route group), so no auth header is needed.
 * This makes it a pure read-throughput test without session overhead.
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL } from '../helpers/config.js';

export const options = {
  thresholds: {
    'http_req_duration{type:read}': ['p(95)<300'],
    'http_req_failed': ['rate<0.01'],
  },
};

const LISTING_QUERIES = ['math', 'history', 'science', 'art', 'music', ''];

export default function () {
  const query = LISTING_QUERIES[Math.floor(Math.random() * LISTING_QUERIES.length)];
  const url = query
    ? `${BASE_URL}/v1/listings?q=${encodeURIComponent(query)}`
    : `${BASE_URL}/v1/listings`;

  const res = http.get(url, { tags: { type: 'read' } });

  check(res, {
    'listings status 200': (r) => r.status === 200,
  });

  sleep(1);
}
