package tracing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestTracingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()

	t.Run("BasicMiddleware", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middleware := NewTracingMiddleware(tracer, &logger, DefaultMiddlewareConfig())

		router := gin.New()
		router.Use(middleware.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check trace headers
		if w.Header().Get("x-trace-id") == "" {
			t.Error("Expected x-trace-id header to be set")
		}
	})

	t.Run("IgnoredPaths", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middlewareConfig := DefaultMiddlewareConfig()
		middlewareConfig.IgnorePaths = []string{"/health", "/metrics"}

		middleware := NewTracingMiddleware(tracer, &logger, middlewareConfig)

		router := gin.New()
		router.Use(middleware.Middleware())
		router.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Trace headers should not be set for ignored paths
		if w.Header().Get("x-trace-id") != "" {
			t.Error("Expected no trace headers for ignored path")
		}
	})

	t.Run("IgnoredUserAgents", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middlewareConfig := DefaultMiddlewareConfig()
		middlewareConfig.IgnoreUserAgents = []string{"HealthChecker"}

		middleware := NewTracingMiddleware(tracer, &logger, middlewareConfig)

		router := gin.New()
		router.Use(middleware.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "HealthChecker/1.0")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middleware := NewTracingMiddleware(tracer, &logger, DefaultMiddlewareConfig())

		router := gin.New()
		router.Use(middleware.Middleware())
		router.GET("/error", func(c *gin.Context) {
			c.Error(gin.Error{Err: http.ErrAbortHandler, Type: gin.ErrorTypePrivate})
			c.JSON(500, gin.H{"error": "internal error"})
		})

		req := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 500 {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("GetSpanFromContext", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middleware := NewTracingMiddleware(tracer, &logger, DefaultMiddlewareConfig())

		var capturedSpan *Span

		router := gin.New()
		router.Use(middleware.Middleware())
		router.GET("/test", func(c *gin.Context) {
			capturedSpan = GetSpan(c)
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if capturedSpan == nil {
			t.Error("Expected span to be available in context")
		}
	})

	t.Run("SetAttributeInHandler", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middleware := NewTracingMiddleware(tracer, &logger, DefaultMiddlewareConfig())

		router := gin.New()
		router.Use(middleware.Middleware())
		router.GET("/test", func(c *gin.Context) {
			SetAttribute(c, "custom.key", "custom.value")
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("AddEventInHandler", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middleware := NewTracingMiddleware(tracer, &logger, DefaultMiddlewareConfig())

		router := gin.New()
		router.Use(middleware.Middleware())
		router.GET("/test", func(c *gin.Context) {
			AddEvent(c, "custom-event", map[string]interface{}{
				"event.key": "event.value",
			})
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("SetErrorInHandler", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middleware := NewTracingMiddleware(tracer, &logger, DefaultMiddlewareConfig())

		router := gin.New()
		router.Use(middleware.Middleware())
		router.GET("/test", func(c *gin.Context) {
			SetError(c, http.ErrAbortHandler)
			c.JSON(500, gin.H{"error": "test error"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 500 {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestMiddlewareConfigurations(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("DevelopmentTracing", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middleware := DevelopmentTracing(tracer, &logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ProductionTracing", func(t *testing.T) {
		config := ProductionConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middleware := ProductionTracing(tracer, &logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("TracingWithDefaultConfig", func(t *testing.T) {
		config := DefaultConfig()
		tracer, _ := NewTracer(config, &logger)
		defer tracer.Shutdown(nil)

		middleware := Tracing(tracer, &logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestMiddlewareStats(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(nil)

	middleware := NewTracingMiddleware(tracer, &logger, DefaultMiddlewareConfig())

	stats := middleware.GetStats()
	if stats == nil {
		t.Error("Expected stats, got nil")
	}
}

func TestStartClientSpan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()
	config := DefaultConfig()
	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(nil)

	// Initialize global tracer for StartClientSpan
	InitGlobalTracer(config, &logger)

	middleware := NewTracingMiddleware(tracer, &logger, DefaultMiddlewareConfig())

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/test", func(c *gin.Context) {
		clientCtx, clientSpan := StartClientSpan(c, "external-api-call")
		if clientSpan != nil {
			GlobalFinishSpan(clientSpan)
		}
		if clientCtx == nil {
			t.Error("Expected client context, got nil")
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestResponseWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CaptureResponseBody", func(t *testing.T) {
		// Create a gin context with response writer
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		rw := &responseWriter{
			ResponseWriter: c.Writer,
			body:          make([]byte, 0),
		}

		testData := []byte("test response")
		n, err := rw.Write(testData)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if n != len(testData) {
			t.Errorf("Expected %d bytes written, got %d", len(testData), n)
		}

		if string(rw.body) != string(testData) {
			t.Errorf("Expected body %s, got %s", testData, rw.body)
		}
	})
}

func TestMiddlewareWithNilConfig(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	tracer, _ := NewTracer(config, &logger)
	defer tracer.Shutdown(nil)

	middleware := NewTracingMiddleware(tracer, &logger, nil)

	if middleware == nil {
		t.Fatal("Expected middleware, got nil")
	}
}

func TestMiddlewareWithNilLogger(t *testing.T) {
	config := DefaultConfig()
	tracer, _ := NewTracer(config, nil)
	defer tracer.Shutdown(nil)

	middleware := NewTracingMiddleware(tracer, nil, DefaultMiddlewareConfig())

	if middleware == nil {
		t.Fatal("Expected middleware, got nil")
	}
}
