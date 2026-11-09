import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '20s', target: 50 },
    { duration: '40s', target: 150 },
    { duration: '1m', target: 300 }, // Peak stress to probe server breaking point
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(99)<1000'], // Under stress, 99% must respond within 1 second
  },
};

const BASE_URL = __ENV.API_URL || 'http://127.0.0.1:8080';

export default function () {
  const params = {
    headers: {
      'Host': 'ecom.local',
      'User-Agent': 'k6-stress-runner/1.0',
    },
  };

  const res = http.get(`${BASE_URL}/`, params);
  check(res, {
    'response status is valid (200 or 429)': (r) => r.status === 200 || r.status === 429,
  });

  sleep(0.1);
}
