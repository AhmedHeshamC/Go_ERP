// Peak Load Test - 1000 RPS for 5 minutes
// Tests system performance under peak production load
// Requirements: 17.1, 17.2, 17.3

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const requestCount = new Counter('requests');
const activeUsers = new Gauge('active_users');

// Test configuration
export const options = {
  scenarios: {
    peak_load: {
      executor: 'constant-arrival-rate',
      rate: 1000, // 1000 RPS
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 200,
      maxVUs: 500,
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'], // p95 < 500ms, p99 < 1s
    'http_req_failed': ['rate<0.001'], // Error rate < 0.1%
    'errors': ['rate<0.001'],
    'http_reqs': ['rate>900'], // Maintain at least 900 RPS
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data
let authTokens = [];
let productIds = [];
let orderIds = [];

// Setup function
export function setup() {
  console.log('Setting up peak load test...');
  
  // Create multiple test users
  const tokens = [];
  for (let i = 0; i < 20; i++) {
    const email = `peakuser${i}@example.com`;
    const password = 'password123';
    
    // Register user
    const registerPayload = JSON.stringify({
      first_name: `PeakUser${i}`,
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
      const body = JSON.parse(loginRes.body);
      tokens.push(body.token);
    }
  }
  
  // Create test products
  const products = [];
  if (tokens.length > 0) {
    const adminToken = tokens[0];
    for (let i = 0; i < 50; i++) {
      const productPayload = JSON.stringify({
        name: `Peak Test Product ${i}`,
        sku: `PEAK-${i}-${Date.now()}`,
        description: 'Product for peak load testing',
        price: 29.99 + i,
        cost: 15.50,
        weight: 1.5,
        is_active: true,
      });
      
      const productRes = http.post(`${BASE_URL}/api/v1/products`, productPayload, {
        headers: {
          'Authorization': `Bearer ${adminToken}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (productRes.status === 201) {
        const body = JSON.parse(productRes.body);
        products.push(body.id);
      }
    }
  }
  
  console.log(`Setup complete. Created ${tokens.length} users and ${products.length} products.`);
  return { tokens, products };
}

// Main test function
export default function(data) {
  activeUsers.add(1);
  
  const token = data.tokens[Math.floor(Math.random() * data.tokens.length)];
  const productId = data.products[Math.floor(Math.random() * data.products.length)];
  
  // Realistic workload distribution
  const rand = Math.random();
  
  if (rand < 0.40) {
    // 40% - Product browsing
    testProductBrowsing(token, productId);
  } else if (rand < 0.70) {
    // 30% - Product search
    testProductSearch(token);
  } else if (rand < 0.85) {
    // 15% - Order viewing
    testOrderViewing(token);
  } else if (rand < 0.95) {
    // 10% - Authentication operations
    testAuthOperations();
  } else {
    // 5% - Order creation
    testOrderCreation(token, productId);
  }
  
  // Realistic think time
  sleep(Math.random() * 0.3 + 0.05); // 50-350ms
  
  activeUsers.add(-1);
}

function testProductBrowsing(token, productId) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  // Get products list
  const listRes = http.get(`${BASE_URL}/api/v1/products?limit=20&page=1`, { headers });
  
  requestCount.add(1);
  responseTime.add(listRes.timings.duration);
  
  const listSuccess = check(listRes, {
    'product list status is 200': (r) => r.status === 200,
    'product list response time < 500ms': (r) => r.timings.duration < 500,
    'product list has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data && Array.isArray(body.data);
      } catch (e) {
        return false;
      }
    },
  });
  
  if (!listSuccess) {
    errorRate.add(1);
  }
  
  // Get product details
  if (productId) {
    sleep(0.1); // Think time
    
    const detailRes = http.get(`${BASE_URL}/api/v1/products/${productId}`, { headers });
    
    requestCount.add(1);
    responseTime.add(detailRes.timings.duration);
    
    const detailSuccess = check(detailRes, {
      'product detail status is 200': (r) => r.status === 200,
      'product detail response time < 300ms': (r) => r.timings.duration < 300,
    });
    
    if (!detailSuccess) {
      errorRate.add(1);
    }
  }
}

function testProductSearch(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  const searchTerms = ['laptop', 'phone', 'tablet', 'computer', 'electronics', 'product', 'test'];
  const term = searchTerms[Math.floor(Math.random() * searchTerms.length)];
  
  const searchRes = http.get(`${BASE_URL}/api/v1/products/search?q=${term}&limit=20`, { headers });
  
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

function testOrderViewing(token) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
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

function testAuthOperations() {
  // Test login with existing user
  const userIndex = Math.floor(Math.random() * 20);
  const loginPayload = JSON.stringify({
    email: `peakuser${userIndex}@example.com`,
    password: 'password123',
  });
  
  const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  requestCount.add(1);
  responseTime.add(loginRes.timings.duration);
  
  const success = check(loginRes, {
    'login status is 200': (r) => r.status === 200,
    'login response time < 500ms': (r) => r.timings.duration < 500,
    'login returns token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.token && body.token.length > 0;
      } catch (e) {
        return false;
      }
    },
  });
  
  if (!success) {
    errorRate.add(1);
  }
}

function testOrderCreation(token, productId) {
  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };
  
  if (!productId) {
    return;
  }
  
  const orderPayload = JSON.stringify({
    shipping_method: 'standard',
    currency: 'USD',
    items: [
      {
        product_id: productId,
        quantity: Math.floor(Math.random() * 3) + 1,
        unit_price: 29.99,
      },
    ],
  });
  
  const orderRes = http.post(`${BASE_URL}/api/v1/orders`, orderPayload, { headers });
  
  requestCount.add(1);
  responseTime.add(orderRes.timings.duration);
  
  const success = check(orderRes, {
    'order creation status is 201 or 200': (r) => r.status === 201 || r.status === 200,
    'order creation response time < 1000ms': (r) => r.timings.duration < 1000,
  });
  
  if (!success) {
    errorRate.add(1);
  }
}

// Teardown function
export function teardown(data) {
  console.log('Peak load test complete.');
}

// Handle summary
export function handleSummary(data) {
  const passed = Object.values(data.thresholds || {}).every(t => t.ok);
  
  let summary = '\n';
  summary += '='.repeat(60) + '\n';
  summary += 'Peak Load Test Results (1000 RPS)\n';
  summary += '='.repeat(60) + '\n\n';
  
  // Overall status
  summary += `Status: ${passed ? '✓ PASSED' : '✗ FAILED'}\n\n`;
  
  // Test duration
  const duration = data.state.testRunDurationMs / 1000;
  summary += `Duration: ${duration.toFixed(2)}s\n\n`;
  
  // HTTP metrics
  if (data.metrics.http_reqs) {
    summary += 'HTTP Requests:\n';
    summary += `  Total: ${data.metrics.http_reqs.values.count}\n`;
    summary += `  Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)} req/s\n\n`;
  }
  
  // Response time metrics
  if (data.metrics.http_req_duration) {
    summary += 'Response Time:\n';
    summary += `  Avg: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
    summary += `  Min: ${data.metrics.http_req_duration.values.min.toFixed(2)}ms\n`;
    summary += `  Max: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms\n`;
    summary += `  P50: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms\n`;
    summary += `  P95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms `;
    summary += `${data.metrics.http_req_duration.values['p(95)'] < 500 ? '✓' : '✗'}\n`;
    summary += `  P99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms `;
    summary += `${data.metrics.http_req_duration.values['p(99)'] < 1000 ? '✓' : '✗'}\n\n`;
  }
  
  // Error rate
  if (data.metrics.http_req_failed) {
    const errorRate = data.metrics.http_req_failed.values.rate * 100;
    summary += `Error Rate: ${errorRate.toFixed(3)}% `;
    summary += `${errorRate < 0.1 ? '✓' : '✗'}\n\n`;
  }
  
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
    'tests/load/results/peak-load-test-results.json': JSON.stringify(data, null, 2),
  };
}
