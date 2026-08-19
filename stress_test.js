import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  // stages: [
  //   { duration: '30s', target: 10 },   // Warm-up: 10 concurrent users
  //   { duration: '1m',  target: 100 },  // Ramp-up: 100 concurrent users
  //   { duration: '1m',  target: 500 },  // Stress: 500 concurrent users
  //   { duration: '30s', target: 0 },    // Cool-down
  // ],
  // thresholds: {
  //   http_req_failed: ['rate<0.01'],    // Error rate must be under 1%
  //   http_req_duration: ['p(95)<20'],  // 95% of requests should complete within 20ms
  // },
  vus: 50,
  duration: '1m',
};

export default function () {
  const BASE_URL = 'http://localhost:8080';
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Device-Token': `device-token-${Math.floor(Math.random() * 50)}`, // Simulates 50 different devices
    },
  };

  // 1. POST: Generate short URL (Triggers Redis counter + Cassandra dual writes)
  const payload = JSON.stringify({
    long_url: `https://example.com/item/${Math.floor(Math.random() * 1000000)}`,
  });

  const postRes = http.post(`${BASE_URL}/api/shorten`, payload, params);
  
  check(postRes, {
    'POST status is 200/201': (r) => r.status === 200 || r.status === 201,
  });

  // Extract short code if creation succeeded
  let shortCode = "1000001";
  if (postRes.status === 200 || postRes.status === 201) {
    try {
      shortCode = JSON.parse(postRes.body).short_code;
    } catch (e) {
      // Fallback
    }
  }

  // 2. GET: Read short URL (Tests Redis Cache first, then Cassandra miss)
  const getRes = http.get(http.url`${BASE_URL}/${shortCode}`, params);
  
  check(getRes, {
    'GET status is 200': (r) => r.status === 200,
  });

  sleep(0.05); // Tiny pause between request cycles
}