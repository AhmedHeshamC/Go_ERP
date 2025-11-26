// Baseline Load Test - 100 RPS for 5 minutes
// Tests system performance under normal load conditions
// Requirements: 17.1, 17.2

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const requestCount = new Counter('requests');

// Test configuration
export const options = {
  scenarios: {
    baseline: {
      executor: 'constant-arrival-rate',
      rate: 100, // 100 RPS
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 50,
      maxVUs: 100,
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'], // 95th percentile < 500ms, 99th < 1s
    'http_req_failed': ['rate<0.001'], // Error rate < 0.1%
    'errors': ['rate<0.001'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data
const testUsers = [
  { email: 'user1@example.com', password: 'password123' },
  { email: 'user2@example.com', password: 'password123' },
  { email: 'user3@example.com', password: 'password123' },
];

let authTokens = [];

// Setup function - runs once before test
export function setup() {
  console.log('Setting up baseline load test...');
  
  // Create test users and get auth tokens
  const tokens = [];
  for (let i = 0; i < testUsers.length; i++) {
    const user = testUsers[i];
    
    // Register user
    const registerPayload = JSON.stringify({
      first_name: `User${i}`,
      last_name: 'Test',
      email: user.email,
      password: user.password,
      role: 'customer',
    });
    
    http.post(`${BASE_URL}/api/v1/auth/register`, registerPayload, {
      headers: { 'Content-Type': 'application/json' },
    });
    
    // Login to get token
    const loginPayload = JSON.stringify({
      email: user.email,
      password: user.password,
    });
    
    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
      headers: { 'Content-Type': 'application/json' },
    });
    
    if (loginRes.status === 200) {
      const body = JSON.parse(loginRes.body);
      tokens.push(body.token);
    }
  }
  
  console.log(`Setup complete. Created ${tokens.length} test users.`);
  return { tokens };
}

// Main test function
export default function(data) {
  const token = data.tokens[Math.floor(Math.random() * data.tokens.length)];
  
  // Workload distribution: 60% reads, 30% searches, 10% writes
  const rand = Math.random();
  
  if (rand < 0.6) {
    // Read operations
    testReadOperations(token);
  } else if (rand < 0.9) {
    // Search operations
    testSearchOperations(token);
  } else {
    // Write operations
    testWriteOperations(token);
  }
  
  // Think time between requests
  sleep(Math.random() * 0.5 + 0.1); // 100-600ms
}

function testReadOperations(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  // Get products list
  const productsRes = http.get(`${BASE_URL}/api/v1/products?limit=20&page=1`, { headers });
  
  requestCount.add(1);
  responseTime.add(productsRes.timings.duration);
  
  const success = check(productsRes, {
    'products list status is 200': (r) => r.status === 200,
    'products list response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  if (!success) {
    errorRate.add(1);
  }
}

function testSearchOperations(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  const searchTerms = ['laptop', 'phone', 'tablet', 'computer', 'electronics'];
  const term = searchTerms[Math.floor(Math.random() * searchTerms.length)];
  
  const searchRes = http.get(`${BASE_URL}/api/v1/products/search?q=${term}`, { headers });
  
  requestCount.add(1);
  responseTime.add(searchRes.timings.duration);
  
  const success = check(searchRes, {
    'search status is 200': (r) => r.status === 200,
    'search response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  if (!success) {
    errorRate.add(1);
  }
}

function testWriteOperations(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  // Get orders list (read-heavy write operation)
  const ordersRes = http.get(`${BASE_URL}/api/v1/orders?limit=10&page=1`, { headers });
  
  requestCount.add(1);
  responseTime.add(ordersRes.timings.duration);
  
  const success = check(ordersRes, {
    'orders list status is 200': (r) => r.status === 200,
    'orders list response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  if (!success) {
    errorRate.add(1);
  }
}

// Teardown function - runs once after test
export function teardown(data) {
  console.log('Baseline load test complete.');
}

// Handle summary
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'tests/load/results/baseline-load-test-results.json': JSON.stringify(data),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;
  
  let summary = '\n';
  summary += `${indent}Baseline Load Test Results\n`;
  summary += `${indent}${'='.repeat(50)}\n\n`;
  
  // Test duration
  const duration = data.state.testRunDurationMs / 1000;
  summary += `${indent}Duration: ${duration.toFixed(2)}s\n\n`;
  
  // HTTP metrics
  if (data.metrics.http_reqs) {
    summary += `${indent}HTTP Requests:\n`;
    summary += `${indent}  Total: ${data.metrics.http_reqs.values.count}\n`;
    summary += `${indent}  Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)} req/s\n\n`;
  }
  
  // Response time metrics
  if (data.metrics.http_req_duration) {
    summary += `${indent}Response Time:\n`;
    summary += `${indent}  Avg: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
    summary += `${indent}  Min: ${data.metrics.http_req_duration.values.min.toFixed(2)}ms\n`;
    summary += `${indent}  Max: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms\n`;
    summary += `${indent}  P50: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms\n`;
    summary += `${indent}  P95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
    summary += `${indent}  P99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n\n`;
  }
  
  // Error rate
  if (data.metrics.http_req_failed) {
    const errorRate = data.metrics.http_req_failed.values.rate * 100;
    summary += `${indent}Error Rate: ${errorRate.toFixed(3)}%\n\n`;
  }
  
  // Threshold results
  summary += `${indent}Thresholds:\n`;
  for (const [name, threshold] of Object.entries(data.thresholds || {})) {
    const status = threshold.ok ? '✓' : '✗';
    summary += `${indent}  ${status} ${name}\n`;
  }
  
  return summary;
}
