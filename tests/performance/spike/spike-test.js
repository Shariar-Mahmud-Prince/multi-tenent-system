import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 10 },
    { duration: '10s', target: 250 }, // Instant massive traffic burst
    { duration: '30s', target: 250 }, // Hold burst
    { duration: '10s', target: 10 },  // Quick drop to evaluate recovery time
  ],
};

const BASE_URL = __ENV.API_URL || 'http://127.0.0.1:8080';

export default function () {
  const params = {
    headers: {
      'Host': 'ecom.local',
      'User-Agent': 'k6-spike-simulator/1.0',
    },
  };

  const res = http.get(`${BASE_URL}/`, params);
  check(res, {
    'server survived spike (not 5xx)': (r) => r.status < 500,
  });

  sleep(0.05);
}
