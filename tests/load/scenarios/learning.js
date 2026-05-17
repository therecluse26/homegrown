/**
 * Learning scenario — exercises GET /v1/learning/students/:id/activities.
 *
 * setup() logs in once and fetches the first student ID so VUs share a stable
 * session token and don't hammer the login endpoint.
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL } from '../helpers/config.js';
import { login, fetchFirstStudentId, authParams } from '../helpers/auth.js';

export const options = {
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
  if (!data.studentId) {
    // No student in seed data — skip without failing.
    return;
  }

  const res = http.get(
    `${BASE_URL}/v1/learning/students/${data.studentId}/activities`,
    authParams(data.sessionToken, 'read'),
  );

  check(res, {
    'activities status 200': (r) => r.status === 200,
  });

  sleep(1);
}
