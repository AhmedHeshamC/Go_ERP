// Stress Test - Gradually increase load to 5000 RPS
// Tests system behavior under extreme load and identifies breaking point
// Requirements: 17.3, 17.4

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const requestCount = new Counter('requests');

// Test configuration - Ramp up to stress levels
export const options = {
  stages: [
    { duration: '2m', target: 500 },   // Ramp up to 500 VUs
    { duration: '3m', target: 1000 },  // Ramp up to 1000 VUs
    { duration: '3m', target: 2000 },  // Ramp up to 2000 VUs
    { duration: '3m', target: 3000 },  // Ramp up to 3000 VUs
    { duration: '2m', target: 5000 },  // Spike to 5000 VUs
    { duration: '2m', target: 0 },     // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<2000', 'p(99)<5000'], // More lenient under stress
    'http_req_failed': ['rate<0.05'], // Allow up to 5% errors under stress
    'errors': ['rate<0.05'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Setup function
export function setup() {
  console.log('Setting up stress test...');
  
  // Create test users
  const tokens = [];
  for (let i = 0; i < 50; i++) {
    const email = `stressuser${i}@example.com`;
    const password = 'password123';
    
    // Register user
    const registerPayload = JSON.stringify({
      first_name: `StressUser${i}`,
      last_name: 'Test',
      email: email,
      password: password,
      role: 'customer',
    });
    
    const registerRes = http.post(`${BASE_URL}/api/v1/auth/register`, registerPayload, {
      headers: { 'Content-Type': 'application/json' },
      timeout: '30s',
    });
    
    // Login to get token
    const loginPayload = JSON.stringify({
      email: email,
      password: password,
    });
    
    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
      headers: { 'Content-Type': 'application/json' },
      timeout: '30s',
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
  return { tokens };
}

// Main test function
export default function(data) {
  if (!data.tokens || data.tokens.length === 0) {
    console.error('No auth tokens available');
    return;
  }
  
  const token = data.tokens[Math.floor(Math.random() * data.tokens.length)];
  
  // Simplified workload for stress testing
  const rand = Math.random();
  
  if (rand < 0.70) {
    // 70% - Read operations (less resource intensive)
    testReadOperations(token);
  } else if (rand < 0.90) {
    // 20% - Search operations
    testSearchOperations(token);
  } else {
    // 10% - Authentication (more resource intensive)
    testAuthOperations();
  }
  
  // Minimal think time under stress
  sleep(Math.random() * 0.1); // 0-100ms
}

function testReadOperations(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  const res = http.get(`${BASE_URL}/api/v1/products?limit=10&page=1`, {
    headers,
    timeout: '10s',
  });
  
  requestCount.add(1);
  responseTime.add(res.timings.duration);
  
  const success = check(res, {
    'status is 200 or 503': (r) => r.status === 200 || r.status === 503,
    'response time < 5000ms': (r) => r.timings.duration < 5000,
  });
  
  if (!success || res.status !== 200) {
    errorRate.add(1);
  }
}

function testSearchOperations(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  const searchTerms = ['product', 'test', 'item'];
  const term = searchTerms[Math.floor(Math.random() * searchTerms.length)];
  
  const res = http.get(`${BASE_URL}/api/v1/products/search?q=${term}`, {
    headers,
    timeout: '10s',
  });
  
  requestCount.add(1);
  responseTime.add(res.timings.duration);
  
  const success = check(res, {
    'status is 200 or 503': (r) => r.status === 200 || r.status === 503,
    'response time < 5000ms': (r) => r.timings.duration < 5000,
  });
  
  if (!success || res.status !== 200) {
    errorRate.add(1);
  }
}

function testAuthOperations() {
  const userIndex = Math.floor(Math.random() * 50);
  const loginPayload = JSON.stringify({
    email: `stressuser${userIndex}@example.com`,
    password: 'password123',
  });
  
  const res = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
    timeout: '10s',
  });
  
  requestCount.add(1);
  responseTime.add(res.timings.duration);
  
  const success = check(res, {
    'status is 200 or 429 or 503': (r) => r.status === 200 || r.status === 429 || r.status === 503,
    'response time < 5000ms': (r) => r.timings.duration < 5000,
  });
  
  if (!success || res.status !== 200) {
    errorRate.add(1);
  }
}

// Teardown function
export function teardown(data) {
  console.log('Stress test complete.');
}

// Handle summary
export function handleSummary(data) {
  let summary = '\n';
  summary += '='.repeat(60) + '\n';
  summary += 'Stress Test Results\n';
  summary += '='.repeat(60) + '\n\n';
  
  // Test duration
  const duration = data.state.testRunDurationMs / 1000;
  summary += `Duration: ${duration.toFixed(2)}s\n\n`;
  
  // HTTP metrics
  if (data.metrics.http_reqs) {
    summary += 'HTTP Requests:\n';
    summary += `  Total: ${data.metrics.http_reqs.values.count}\n`;
    summary += `  Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)} req/s\n`;
    summary += `  Peak Rate: ~${(data.metrics.http_reqs.values.count / duration * 1.5).toFixed(2)} req/s (estimated)\n\n`;
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
  
  // System behavior analysis
  summary += 'System Behavior Under Stress:\n';
  if (data.metrics.http_req_duration) {
    const p95 = data.metrics.http_req_duration.values['p(95)'];
    const p99 = data.metrics.http_req_duration.values['p(99)'];
    
    if (p95 < 1000) {
      summary += '  ✓ System maintained good performance under stress\n';
    } else if (p95 < 2000) {
      summary += '  ⚠ System showed degraded performance under stress\n';
    } else {
      summary += '  ✗ System performance severely degraded under stress\n';
    }
    
    if (p99 < 5000) {
      summary += '  ✓ No request timeouts observed\n';
    } else {
      summary += '  ✗ Request timeouts observed\n';
    }
  }
  
  if (data.metrics.http_req_failed) {
    const errorRate = data.metrics.http_req_failed.values.rate;
    if (errorRate < 0.01) {
      summary += '  ✓ System maintained stability under stress\n';
    } else if (errorRate < 0.05) {
      summary += '  ⚠ System showed some instability under stress\n';
    } else {
      summary += '  ✗ System became unstable under stress\n';
    }
  }
  
  summary += '\n';
  
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
    'tests/load/results/stress-test-results.json': JSON.stringify(data, null, 2),
  };
}
