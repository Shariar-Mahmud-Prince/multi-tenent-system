import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 }, // Ramp up to 20 users
    { duration: '1m', target: 50 },  // Sustained load with 50 concurrent users
    { duration: '20s', target: 0 },  // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<300'], // 95% of requests must complete below 300ms
    http_req_failed: ['rate<0.01'],    // Error rate must stay below 1%
  },
};

const BASE_URL = __ENV.API_URL || 'http://127.0.0.1:8080';

export default function () {
  // Test storefront routing with tenant header
  const storeParams = {
    headers: {
      'Host': 'ecom.local',
      'User-Agent': 'k6-load-tester/1.0',
    },
  };
  const storeRes = http.get(`${BASE_URL}/`, storeParams);
  check(storeRes, {
    'storefront status is 200': (r) => r.status === 200,
    'storefront returns html/json': (r) => r.body.length > 0,
  });

  // Test system health
  const healthRes = http.get(`${BASE_URL}/healthz`);
  check(healthRes, {
    'healthz status is 200': (r) => r.status === 200,
  });

  // Test TLS validation endpoint
  const validateRes = http.get(`${BASE_URL}/api/v1/domains/validate?domain=ecom.local`);
  check(validateRes, {
    'validate domain status is 200': (r) => r.status === 200,
  });

  sleep(0.5);
}
