// Shared configuration for all load test scenarios.
// Override via k6 environment variables: BASE_URL, KRATOS_URL, EMAIL, PASSWORD.
export const BASE_URL = __ENV.BASE_URL || 'http://localhost:3500';
export const KRATOS_URL = __ENV.KRATOS_URL || 'http://localhost:4935';
export const TEST_EMAIL = __ENV.TEST_EMAIL || 'seed@example.com';
export const TEST_PASSWORD = __ENV.TEST_PASSWORD || 'SeedPassword123!';

// p95 ≤ 300 ms target for all read endpoints [HOM-22].
export const p95Threshold = { 'http_req_duration{type:read}': ['p(95)<300'] };
