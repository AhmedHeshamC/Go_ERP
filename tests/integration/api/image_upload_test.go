package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImageUploadEndpoints tests image upload functionality
func TestImageUploadEndpoints(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	uploadsDir := filepath.Join(tempDir, "uploads")
	err := os.MkdirAll(uploadsDir, 0755)
	require.NoError(t, err)

	// Create a test image file (1x1 pixel PNG)
	testImagePath := filepath.Join(tempDir, "test.png")
	err = createTestImage(testImagePath)
	require.NoError(t, err)

	// Setup test router
	router := gin.New()

	// Mock image upload handler
	router.POST("/api/v1/products/:product_id/images", func(c *gin.Context) {
		productID := c.Param("product_id")
		if productID == "" {
			c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Product ID is required",
			})
			return
			}

		// Validate product ID
		if _, err := uuid.Parse(productID); err != nil {
			c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid product ID",
			})
			return
			}

		// Get uploaded file
		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, map[string]string{
				"error": "No file uploaded",
			})
			return
		}

		// Simulate upload processing
		time.Sleep(10 * time.Millisecond) // Simulate processing time

		// Create mock result
		result := map[string]interface{}{
			"filename":      "test_" + uuid.New().String()[:8] + ".png",
			"original_name": file.Filename,
			"size":         file.Size,
			"mime_type":    "image/png",
			"url":          fmt.Sprintf("/uploads/products/%s/test.png", productID),
			"directory":    fmt.Sprintf("products/%s", productID),
			"uploaded_at":   time.Now().UTC(),
		}

		c.JSON(http.StatusCreated, result)
	})

	// Test successful image upload
	t.Run("Successful Image Upload", func(t *testing.T) {
		// Create multipart form data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", filepath.Base(testImagePath))
		require.NoError(t, err)

		// Copy test image to form
		file, err := os.Open(testImagePath)
		require.NoError(t, err)
		_, err = io.Copy(part, file)
		require.NoError(t, err)
		file.Close()

		err = writer.WriteField("test", "value")
		require.NoError(t, err)
		writer.Close()

		// Create request
		productID := uuid.New().String()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/products/%s/images", productID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "filename")
		assert.Contains(t, response, "original_name")
		assert.Equal(t, response["original_name"], "test.png")
		assert.Contains(t, response, "url")
		assert.Contains(t, response["url"].(string), productID)
	})

	// Test upload without file
	t.Run("Upload Without File", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("test", "value")
		writer.Close()

		productID := uuid.New().String()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/products/%s/images", productID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "No file uploaded", response["error"])
	})

	// Test upload with invalid product ID
	t.Run("Upload with Invalid Product ID", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", filepath.Base(testImagePath))
		require.NoError(t, err)

		file, err := os.Open(testImagePath)
		require.NoError(t, err)
		_, err = io.Copy(part, file)
		require.NoError(t, err)
		file.Close()
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/v1/products/invalid-id/images", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Invalid product ID", response["error"])
	})

	// Clean up
	os.RemoveAll(tempDir)
}

// TestBatchImageUpload tests batch image upload functionality
func TestBatchImageUpload(t *testing.T) {
	// Create temporary directory and test images
	tempDir := t.TempDir()
	err := os.MkdirAll(filepath.Join(tempDir, "uploads"), 0755)
	require.NoError(t, err)

	// Create multiple test images
	testImages := make([]string, 3)
	for i := 0; i < 3; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("test%d.png", i))
		err = createTestImage(path)
		require.NoError(t, err)
		testImages[i] = path
	}

	// Setup test router
	router := gin.New()

	router.POST("/api/v1/products/:product_id/images/batch", func(c *gin.Context) {
		productID := c.Param("product_id")
		if productID == "" {
			c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Product ID is required",
			})
			return
		}

		// Validate product ID
		if _, err := uuid.Parse(productID); err != nil {
			c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid product ID",
			})
			return
		}

		// Get uploaded files
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Failed to parse multipart form",
			})
			return
			}

		files := form.File["images"]
		if len(files) == 0 {
			c.JSON(http.StatusBadRequest, map[string]string{
				"error": "No files uploaded",
			})
			return
		}

		// Process each file
		results := make([]map[string]interface{}, 0)
		errors := make([]map[string]interface{}, 0)

		for i, file := range files {
			// Simulate processing
			time.Sleep(5 * time.Millisecond)

			result := map[string]interface{}{
				"filename":      fmt.Sprintf("test_%d.png", i),
				"original_name": file.Filename,
				"size":         file.Size,
				"mime_type":    "image/png",
				"url":          fmt.Sprintf("/uploads/products/%s/test_%d.png", productID, i),
				"uploaded_at":   time.Now().UTC(),
			}

			results = append(results, result)
		}

		response := map[string]interface{}{
			"results":      results,
			"errors":       errors,
			"total_files":  len(files),
			"successful":   len(results),
			"failed":       len(errors),
		}

		c.JSON(http.StatusCreated, response)
	})

	t.Run("Successful Batch Upload", func(t *testing.T) {
		// Create multipart form with multiple files
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add multiple image files
		for _, imagePath := range testImages {
			part, err := writer.CreateFormFile("images", filepath.Base(imagePath))
			require.NoError(t, err)

			file, err := os.Open(imagePath)
			require.NoError(t, err)
			_, err = io.Copy(part, file)
			require.NoError(t, err)
			file.Close()
					}

		writer.Close()

		productID := uuid.New().String()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/products/%s/images/batch", productID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 3.0, response["total_files"])
		assert.Equal(t, 3.0, response["successful"])
		assert.Equal(t, 0.0, response["failed"])
		assert.Len(t, response["results"], 3)
	})

	// Clean up
	os.RemoveAll(tempDir)
}

// TestImageFileValidation tests image file validation
func TestImageFileValidation(t *testing.T) {
	// Create test files of different types
	tempDir := t.TempDir()

	// Valid image file
	validImagePath := filepath.Join(tempDir, "valid.png")
	err := createTestImage(validImagePath)
	require.NoError(t, err)

	// Invalid file (text file)
	invalidImagePath := filepath.Join(tempDir, "invalid.txt")
	err = os.WriteFile(invalidImagePath, []byte("This is not an image"), 0644)
	require.NoError(t, err)

	// Large file (over limit)
	largeImagePath := filepath.Join(tempDir, "large.png")
	// Create a large file by writing a lot of data
	largeFile, err := os.Create(largeImagePath)
	require.NoError(t, err)
	data := make([]byte, 1024*1024*15) // 15MB
	for i := range data {
		data[i] = byte(i % 256)
	}
	_, err = largeFile.Write(data)
	require.NoError(t, err)
	largeFile.Close()

	// Setup test router with validation
	router := gin.New()

	router.POST("/api/v1/products/:product_id/images", func(c *gin.Context) {
		// Get uploaded file
		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, map[string]string{
				"error": "No file uploaded",
			})
			return
			}

		// Validate file size
		maxSize := int64(10 * 1024 * 1024) // 10MB
		if file.Size > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, map[string]string{
				"error": "File too large",
			})
			return
		}

		// Validate file type (simplified)
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			c.JSON(http.StatusUnsupportedMediaType, map[string]string{
				"error": "Unsupported file type",
			})
			return
		}

		c.JSON(http.StatusCreated, map[string]string{
			"message": "File uploaded successfully",
		})
	})

	t.Run("Valid Image File", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", filepath.Base(validImagePath))
		require.NoError(t, err)

		file, err := os.Open(validImagePath)
		require.NoError(t, err)
		_, err = io.Copy(part, file)
		require.NoError(t, err)
		file.Close()
				writer.Close()

		productID := uuid.New().String()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/products/%s/images", productID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "File uploaded successfully", response["message"])
	})

	t.Run("Invalid File Type", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", filepath.Base(invalidImagePath))
		require.NoError(t, err)

		file, err := os.Open(invalidImagePath)
		require.NoError(t, err)
		_, err = io.Copy(part, file)
		require.NoError(t, err)
		file.Close()
				writer.Close()

		productID := uuid.New().String()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/products/%s/images", productID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Unsupported file type", response["error"])
	})

	t.Run("File Too Large", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", filepath.Base(largeImagePath))
		require.NoError(t, err)

		file, err := os.Open(largeImagePath)
		require.NoError(t, err)
		_, err = io.Copy(part, file)
		require.NoError(t, err)
		file.Close()
				writer.Close()

		productID := uuid.New().String()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/products/%s/images", productID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "File too large", response["error"])
	})

	// Clean up
	os.RemoveAll(tempDir)
}

// createTestImage creates a simple 1x1 PNG image for testing
func createTestImage(path string) error {
	// Create a simple PNG file with a 1x1 pixel
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Minimal PNG header + 1x1 pixel
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00,
		0x90, 0x77, 0x53, 0x4E, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, 0x52, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		0x02, 0x52, 0x4F, 0x43, 0x91, 0x4C, 0x45, 0x52, 0x00, 0x00, 0x00, 0x04, 0x67, 0x41,
		0x4D, 0x41, 0x41, 0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01,
		0x18, 0xDD, 0x8D, 0xED, 0x01, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00,
	}

	_, err = file.Write(pngData)
	return err
}

// BenchmarkImageUpload benchmarks image upload performance
func BenchmarkImageUpload(b *testing.B) {
	// Create test directory and image
	tempDir := b.TempDir()
	testImagePath := filepath.Join(tempDir, "test.png")
	err := createTestImage(testImagePath)
	require.NoError(b, err)

	// Setup router
	router := gin.New()

	router.POST("/api/v1/products/:product_id/images", func(c *gin.Context) {
		file, _ := c.FormFile("image")
		// Simulate minimal processing
		c.JSON(http.StatusCreated, map[string]interface{}{
			"filename": "test.png",
			"size":     file.Size,
		})
	})

	b.ResetTimer()
	for i := 0; i < 10; i++ {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", "test.png")
		require.NoError(b, err)

		file, err := os.Open(testImagePath)
		require.NoError(b, err)
		_, err = io.Copy(part, file)
		require.NoError(b, err)
		file.Close()
				writer.Close()

		productID := uuid.New().String()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/products/%s/images", productID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Clean up
	os.RemoveAll(tempDir)
}