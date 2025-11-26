// Spike Test - Sudden jump from 100 to 2000 RPS
// Tests system behavior under sudden traffic spikes
// Requirements: 17.3, 17.5

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const requestCount = new Counter('requests');
const recoveryTime = new Trend('recovery_time');

// Test configuration - Sudden spike
export const options = {
  stages: [
    { duration: '1m', target: 100 },   // Normal load
    { duration: '30s', target: 2000 }, // Sudden spike
    { duration: '3m', target: 2000 },  // Sustain spike
    { duration: '1m', target: 100 },   // Return to normal
    { duration: '1m', target: 100 },   // Observe recovery
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000', 'p(99)<2000'], // Lenient during spike
    'http_req_failed': ['rate<0.02'], // Allow up to 2% errors during spike
    'errors': ['rate<0.02'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

let spikeStartTime = null;
let normalPerformanceRestored = false;

// Setup function
export function setup() {
  console.log('Setting up spike test...');
  
  // Create test users
  const tokens = [];
  for (let i = 0; i < 30; i++) {
    const email = `spikeuser${i}@example.com`;
    const password = 'password123';
    
    // Register user
    const registerPayload = JSON.stringify({
      first_name: `SpikeUser${i}`,
      last_name: 'Test',
      email: email,
      password: password,
      role: 'customer',
    });
    
    http.post(`${BASE_URL}/api/v1/auth/register`, registerPayload, {
      headers: { 'Content-Type': 'application/json' },
    });
    
    // Login to get token
    const loginPayload = JSON.stringify({
      email: email,
      password: password,
    });
    
    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
      headers: { 'Content-Type': 'application/json' },
    });
    
    if (loginRes.status === 200) {
      try {
        const body = JSON.parse(loginRes.body);
        tokens.push(body.token);
      } catch (e) {
        console.error(`Failed to parse login response for user ${i}`);
      }
    }
  }
  
  console.log(`Setup complete. Created ${tokens.length} test users.`);
  return { tokens, spikeStartTime: Date.now() };
}

// Main test function
export default function(data) {
  if (!data.tokens || data.tokens.length === 0) {
    console.error('No auth tokens available');
    return;
  }
  
  const token = data.tokens[Math.floor(Math.random() * data.tokens.length)];
  
  // Track spike timing
  const currentVUs = __VU;
  if (currentVUs > 500 && !spikeStartTime) {
    spikeStartTime = Date.now();
  }
  
  // Workload distribution
  const rand = Math.random();
  
  if (rand < 0.50) {
    // 50% - Product reads
    testProductReads(token);
  } else if (rand < 0.80) {
    // 30% - Search
    testSearch(token);
  } else if (rand < 0.95) {
    // 15% - Orders
    testOrders(token);
  } else {
    // 5% - Auth
    testAuth();
  }
  
  // Variable think time based on load
  const thinkTime = currentVUs > 1000 ? 0.05 : 0.2;
  sleep(Math.random() * thinkTime);
}

function testProductReads(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  const res = http.get(`${BASE_URL}/api/v1/products?limit=20&page=1`, {
    headers,
    timeout: '10s',
  });
  
  requestCount.add(1);
  responseTime.add(res.timings.duration);
  
  // Track recovery time
  if (spikeStartTime && res.timings.duration < 500 && !normalPerformanceRestored) {
    const recovery = Date.now() - spikeStartTime;
    recoveryTime.add(recovery);
    normalPerformanceRestored = true;
  }
  
  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 2000ms': (r) => r.timings.duration < 2000,
  });
  
  if (!success) {
    errorRate.add(1);
  }
}

function testSearch(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  const searchTerms = ['laptop', 'phone', 'product'];
  const term = searchTerms[Math.floor(Math.random() * searchTerms.length)];
  
  const res = http.get(`${BASE_URL}/api/v1/products/search?q=${term}`, {
    headers,
    timeout: '10s',
  });
  
  requestCount.add(1);
  responseTime.add(res.timings.duration);
  
  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 2000ms': (r) => r.timings.duration < 2000,
  });
  
  if (!success) {
    errorRate.add(1);
  }
}

function testOrders(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  const res = http.get(`${BASE_URL}/api/v1/orders?limit=10&page=1`, {
    headers,
    timeout: '10s',
  });
  
  requestCount.add(1);
  responseTime.add(res.timings.duration);
  
  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 2000ms': (r) => r.timings.duration < 2000,
  });
  
  if (!success) {
    errorRate.add(1);
  }
}

function testAuth() {
  const userIndex = Math.floor(Math.random() * 30);
  const loginPayload = JSON.stringify({
    email: `spikeuser${userIndex}@example.com`,
    password: 'password123',
  });
  
  const res = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
    timeout: '10s',
  });
  
  requestCount.add(1);
  responseTime.add(res.timings.duration);
  
  const success = check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'response time < 2000ms': (r) => r.timings.duration < 2000,
  });
  
  if (!success || res.status !== 200) {
    errorRate.add(1);
  }
}

// Teardown function
export function teardown(data) {
  console.log('Spike test complete.');
}

// Handle summary
export function handleSummary(data) {
  let summary = '\n';
  summary += '='.repeat(60) + '\n';
  summary += 'Spike Test Results (100 → 2000 RPS)\n';
  summary += '='.repeat(60) + '\n\n';
  
  // Test duration
  const duration = data.state.testRunDurationMs / 1000;
  summary += `Duration: ${duration.toFixed(2)}s\n\n`;
  
  // HTTP metrics
  if (data.metrics.http_reqs) {
    summary += 'HTTP Requests:\n';
    summary += `  Total: ${data.metrics.http_reqs.values.count}\n`;
    summary += `  Average Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)} req/s\n\n`;
  }
  
  // Response time metrics
  if (data.metrics.http_req_duration) {
    summary += 'Response Time:\n';
    summary += `  Avg: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
    summary += `  Min: ${data.metrics.http_req_duration.values.min.toFixed(2)}ms\n`;
    summary += `  Max: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms\n`;
    summary += `  P50: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms\n`;
    summary += `  P95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
    summary += `  P99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n\n`;
  }
  
  // Error rate
  if (data.metrics.http_req_failed) {
    const errorRate = data.metrics.http_req_failed.values.rate * 100;
    summary += `Error Rate: ${errorRate.toFixed(3)}%\n\n`;
  }
  
  // Spike resilience analysis
  summary += 'Spike Resilience Analysis:\n';
  if (data.metrics.http_req_duration) {
    const p95 = data.metrics.http_req_duration.values['p(95)'];
    const p99 = data.metrics.http_req_duration.values['p(99)'];
    
    if (p95 < 500) {
      summary += '  ✓ System handled spike with minimal performance impact\n';
    } else if (p95 < 1000) {
      summary += '  ⚠ System showed moderate performance degradation during spike\n';
    } else {
      summary += '  ✗ System showed significant performance degradation during spike\n';
    }
    
    if (p99 < 2000) {
      summary += '  ✓ System maintained acceptable worst-case performance\n';
    } else {
      summary += '  ✗ System had unacceptable worst-case performance\n';
    }
  }
  
  if (data.metrics.http_req_failed) {
    const errorRate = data.metrics.http_req_failed.values.rate;
    if (errorRate < 0.01) {
      summary += '  ✓ System maintained high reliability during spike\n';
    } else if (errorRate < 0.02) {
      summary += '  ⚠ System showed some reliability issues during spike\n';
    } else {
      summary += '  ✗ System had significant reliability issues during spike\n';
    }
  }
  
  summary += '\n';
  
  // Recommendations
  summary += 'Recommendations:\n';
  if (data.metrics.http_req_duration && data.metrics.http_req_duration.values['p(95)'] > 1000) {
    summary += '  - Consider implementing auto-scaling to handle traffic spikes\n';
    summary += '  - Review connection pool sizing and timeout configurations\n';
  }
  if (data.metrics.http_req_failed && data.metrics.http_req_failed.values.rate > 0.01) {
    summary += '  - Implement circuit breakers to prevent cascade failures\n';
    summary += '  - Add request queuing with backpressure\n';
  }
  summary += '  - Monitor system resources during spikes\n';
  summary += '  - Consider implementing rate limiting for protection\n\n';
  
  // Threshold results
  summary += 'Thresholds:\n';
  for (const [name, threshold] of Object.entries(data.thresholds || {})) {
    const status = threshold.ok ? '✓' : '✗';
    summary += `  ${status} ${name}\n`;
  }
  
  summary += '\n' + '='.repeat(60) + '\n';
  
  console.log(summary);
  
  return {
    'stdout': summary,
    'tests/load/results/spike-test-results.json': JSON.stringify(data, null, 2),
  };
}
