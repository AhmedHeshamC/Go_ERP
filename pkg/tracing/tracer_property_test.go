package tracing

import (
	"context"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"pgregory.net/rapid"
)

// **Feature: production-readiness, Property 15: Trace ID Uniqueness**
// **Validates: Requirements 12.1**
//
// Property 15: Trace ID Uniqueness
// For any two concurrent requests, the trace IDs generated must be unique
func TestProperty_TraceIDUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random number of concurrent requests (between 2 and 100)
		numRequests := rapid.IntRange(2, 100).Draw(t, "num_requests")

		// Create a tracer with default config
		nopLogger := zerolog.Nop()
		config := DefaultConfig()
		tracer, err := NewTracer(config, &nopLogger)
		if err != nil {
			t.Fatalf("failed to create tracer: %v", err)
		}
		defer tracer.Shutdown(context.Background())

		// Use a map to track trace IDs and detect duplicates
		traceIDs := make(map[string]bool)
		var mu sync.Mutex
		var wg sync.WaitGroup

		// Channel to collect any duplicate trace IDs
		duplicates := make(chan string, numRequests)

		// Start multiple concurrent requests
		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(requestNum int) {
				defer wg.Done()

				// Create a new context for this request
				ctx := context.Background()

				// Start a span (which generates a trace ID)
				_, span := tracer.StartSpan(ctx, "test-operation", SpanKindServer)

				// Check if this trace ID is unique
				mu.Lock()
				if traceIDs[span.TraceID] {
					// Duplicate found!
					duplicates <- span.TraceID
				} else {
					traceIDs[span.TraceID] = true
				}
				mu.Unlock()

				// Finish the span
				tracer.FinishSpan(span)
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(duplicates)

		// Property: All trace IDs must be unique (no duplicates)
		var foundDuplicates []string
		for dup := range duplicates {
			foundDuplicates = append(foundDuplicates, dup)
		}

		if len(foundDuplicates) > 0 {
			t.Fatalf("trace ID uniqueness violated: found %d duplicate trace IDs, first duplicate: %s",
				len(foundDuplicates), foundDuplicates[0])
		}

		// Verify we generated the expected number of unique trace IDs
		if len(traceIDs) != numRequests {
			t.Fatalf("expected %d unique trace IDs, but got %d", numRequests, len(traceIDs))
		}
	})
}

// Additional property test: Span ID uniqueness within a trace
// This tests that span IDs are unique even within the same trace
func TestProperty_SpanIDUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random number of spans (between 2 and 50)
		numSpans := rapid.IntRange(2, 50).Draw(t, "num_spans")

		// Create a tracer with default config
		nopLogger := zerolog.Nop()
		config := DefaultConfig()
		tracer, err := NewTracer(config, &nopLogger)
		if err != nil {
			t.Fatalf("failed to create tracer: %v", err)
		}
		defer tracer.Shutdown(context.Background())

		// Create a parent context
		ctx := context.Background()

		// Track all span IDs
		spanIDs := make(map[string]bool)

		// Create multiple spans in the same trace
		for i := 0; i < numSpans; i++ {
			_, span := tracer.StartSpan(ctx, "test-operation", SpanKindInternal)

			// Check if this span ID is unique
			if spanIDs[span.SpanID] {
				t.Fatalf("span ID uniqueness violated: duplicate span ID %s", span.SpanID)
			}
			spanIDs[span.SpanID] = true

			// Finish the span
			tracer.FinishSpan(span)
		}

		// Property: All span IDs must be unique
		if len(spanIDs) != numSpans {
			t.Fatalf("expected %d unique span IDs, but got %d", numSpans, len(spanIDs))
		}
	})
}

// Additional property test: Trace ID format consistency
// This tests that all generated trace IDs follow the expected format
func TestProperty_TraceIDFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random number of trace IDs to test
		numTraceIDs := rapid.IntRange(1, 50).Draw(t, "num_trace_ids")

		// Create a tracer with default config
		nopLogger := zerolog.Nop()
		config := DefaultConfig()
		tracer, err := NewTracer(config, &nopLogger)
		if err != nil {
			t.Fatalf("failed to create tracer: %v", err)
		}
		defer tracer.Shutdown(context.Background())

		for i := 0; i < numTraceIDs; i++ {
			ctx := context.Background()
			_, span := tracer.StartSpan(ctx, "test-operation", SpanKindServer)

			// Property: Trace ID must be non-empty
			if span.TraceID == "" {
				t.Fatalf("trace ID is empty")
			}

			// Property: Trace ID must be a valid hex string (UUID without dashes)
			// Expected length is 32 characters (UUID without dashes)
			if len(span.TraceID) != 32 {
				t.Fatalf("trace ID has invalid length: expected 32, got %d (trace ID: %s)",
					len(span.TraceID), span.TraceID)
			}

			// Property: Trace ID must contain only hex characters
			for _, c := range span.TraceID {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
					t.Fatalf("trace ID contains invalid character '%c': %s", c, span.TraceID)
				}
			}

			tracer.FinishSpan(span)
		}
	})
}

// Additional property test: Concurrent trace ID generation stress test
// This tests trace ID uniqueness under high concurrency
func TestProperty_ConcurrentTraceIDGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a large number of concurrent requests (between 100 and 1000)
		numRequests := rapid.IntRange(100, 1000).Draw(t, "num_requests")

		// Create a tracer with default config
		nopLogger := zerolog.Nop()
		config := DefaultConfig()
		tracer, err := NewTracer(config, &nopLogger)
		if err != nil {
			t.Fatalf("failed to create tracer: %v", err)
		}
		defer tracer.Shutdown(context.Background())

		// Use a concurrent-safe map to track trace IDs
		traceIDs := sync.Map{}
		var wg sync.WaitGroup

		// Counter for duplicates
		var duplicateCount int32

		// Start many concurrent requests
		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				ctx := context.Background()
				_, span := tracer.StartSpan(ctx, "stress-test", SpanKindServer)

				// Try to store the trace ID
				if _, loaded := traceIDs.LoadOrStore(span.TraceID, true); loaded {
					// This trace ID was already seen - duplicate!
					duplicateCount++
				}

				tracer.FinishSpan(span)
			}()
		}

		wg.Wait()

		// Property: No duplicates should be found
		if duplicateCount > 0 {
			t.Fatalf("trace ID uniqueness violated under high concurrency: found %d duplicates out of %d requests",
				duplicateCount, numRequests)
		}

		// Count unique trace IDs
		uniqueCount := 0
		traceIDs.Range(func(key, value interface{}) bool {
			uniqueCount++
			return true
		})

		// Property: Number of unique trace IDs should equal number of requests
		if uniqueCount != numRequests {
			t.Fatalf("expected %d unique trace IDs, but got %d", numRequests, uniqueCount)
		}
	})
}

// Additional property test: Parent-child span relationship
// This tests that child spans have different span IDs but share the same trace ID
func TestProperty_ParentChildSpanRelationship(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random number of child spans (between 1 and 20)
		numChildren := rapid.IntRange(1, 20).Draw(t, "num_children")

		// Create a tracer with default config
		nopLogger := zerolog.Nop()
		config := DefaultConfig()
		tracer, err := NewTracer(config, &nopLogger)
		if err != nil {
			t.Fatalf("failed to create tracer: %v", err)
		}
		defer tracer.Shutdown(context.Background())

		// Create a parent span
		ctx := context.Background()
		parentCtx, parentSpan := tracer.StartSpan(ctx, "parent-operation", SpanKindServer)

		// Track child span IDs
		childSpanIDs := make(map[string]bool)

		// Create multiple child spans
		for i := 0; i < numChildren; i++ {
			_, childSpan := tracer.StartSpan(parentCtx, "child-operation", SpanKindInternal)

			// Property: Child span must have the same trace ID as parent
			if childSpan.TraceID != parentSpan.TraceID {
				t.Fatalf("child span has different trace ID: parent=%s, child=%s",
					parentSpan.TraceID, childSpan.TraceID)
			}

			// Property: Child span must have a different span ID than parent
			if childSpan.SpanID == parentSpan.SpanID {
				t.Fatalf("child span has same span ID as parent: %s", childSpan.SpanID)
			}

			// Property: Child span must have a unique span ID
			if childSpanIDs[childSpan.SpanID] {
				t.Fatalf("duplicate child span ID: %s", childSpan.SpanID)
			}
			childSpanIDs[childSpan.SpanID] = true

			// Property: Child span must reference parent span ID
			if childSpan.ParentSpanID == nil {
				t.Fatalf("child span has no parent span ID")
			}
			if *childSpan.ParentSpanID != parentSpan.SpanID {
				t.Fatalf("child span parent ID mismatch: expected %s, got %s",
					parentSpan.SpanID, *childSpan.ParentSpanID)
			}

			tracer.FinishSpan(childSpan)
		}

		tracer.FinishSpan(parentSpan)

		// Verify all child span IDs are unique
		if len(childSpanIDs) != numChildren {
			t.Fatalf("expected %d unique child span IDs, but got %d", numChildren, len(childSpanIDs))
		}
	})
}
