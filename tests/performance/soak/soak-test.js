import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 30 },
    { duration: '5m', target: 30 }, // Sustained steady endurance load
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.005'], // Error rate must stay strictly under 0.5%
  },
};

const BASE_URL = __ENV.API_URL || 'http://127.0.0.1:8080';

export default function () {
  const params = {
    headers: {
      'Host': 'ecom.local',
      'User-Agent': 'k6-soak-monitor/1.0',
    },
  };

  const res = http.get(`${BASE_URL}/`, params);
  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  sleep(1);
}
