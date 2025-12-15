package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:generate go run tools/generate-embed/main.go

type Config struct {
	Title    string            `json:"title"`
	Mappings []FirmwareMapping `json:"mappings"`
}

type FirmwareMapping struct {
	CustomerType string `json:"customerType"`
	FirmwareType string `json:"firmwareType"`
	FilePath     string `json:"filePath"`
	Embed        bool   `json:"embed"`
	Description  string `json:"description"`
	VersionCode  int    `json:"versionCode"`
	VersionName  string `json:"versionName"`
}

// Pre-calculated firmware details to avoid overhead during requests
type PreCalculatedFirmwareDetail struct {
	MD5         string
	FileExists  bool
	Error       error
}

// Global cache to store pre-calculated data
var firmwareCache map[string]*PreCalculatedFirmwareDetail

type FirmwareResponse struct {
	Success bool           `json:"success"`
	Title   string         `json:"title"`
	Rows    FirmwareDetail `json:"rows"`
}

type FirmwareDetail struct {
	CreateTime   string `json:"createTime"`
	CustomerType string `json:"customerType"`
	Description  string `json:"description"`
	FirmwareMD5  string `json:"firmwareMD5"`
	FirmwareType string `json:"firmwareType"`
	Id           string `json:"id"`
	IsPublish    bool   `json:"is_publish"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	VersionCode  int    `json:"versionCode"`
	VersionName  string `json:"versionName"`
}

func main() {
	bindAddr := flag.String("bind", "localhost", "Bind address for the server (default: localhost)")
	port := flag.String("port", getPort(), "Port for the server (default: 9190 or PORT env var)")
	enableTracing := flag.Bool("trace", false, "Enable request tracing (default: false)")

	flag.Parse()

	config := loadConfig("config.json")

	// Pre-calculate firmware details to avoid overhead during requests
	initializeFirmwareCache(config)

	// Create the base handler
	mux := http.NewServeMux()

	// Endpoint that returns JSON metadata in the expected format
	mux.HandleFunc("/home/downloadnewest", func(w http.ResponseWriter, r *http.Request) {
		customerType := r.URL.Query().Get("customerType")
		firmwareType := r.URL.Query().Get("firmwareType")

		if customerType == "" || firmwareType == "" {
			http.Error(w, "Missing customerType or firmwareType parameters", http.StatusBadRequest)
			return
		}

		mapping := findMapping(config, customerType, firmwareType)
		if mapping == nil {
			http.Error(w, fmt.Sprintf("No mapping found for customerType=%s and firmwareType=%s", customerType, firmwareType), http.StatusNotFound)
			return
		}

		// Get pre-calculated data from cache
		cacheKey := fmt.Sprintf("%s:%s", customerType, firmwareType)
		cachedData, exists := firmwareCache[cacheKey]
		if !exists {
			http.Error(w, "Cache entry not found", http.StatusInternalServerError)
			return
		}

		if cachedData.Error != nil || !cachedData.FileExists {
			http.Error(w, "File not found or error accessing file", http.StatusNotFound)
			return
		}

		// Create response in expected format
		response := FirmwareResponse{
			Success: true,
			Title:   config.Title, // Use title from config
			Rows: FirmwareDetail{
				CreateTime:   time.Now().Format("2006-01-02 15:04:05"), // Current time in format YYYY-MM-DD HH:MM:SS
				CustomerType: customerType,
				Description:  mapping.Description, // Use description from the specific mapping
				FirmwareMD5:  cachedData.MD5,
				FirmwareType: firmwareType,
				Id:           generateId(customerType, firmwareType), // Generate a unique ID
				IsPublish:    true,
				Name:         generateName(customerType, firmwareType),              // Generate a display name
				Path:         "/upload/firmware/" + filepath.Base(mapping.FilePath), // Standardized path format
				VersionCode:  mapping.VersionCode,                                   // Use version code from the specific mapping
				VersionName:  mapping.VersionName,                                   // Use version name from the specific mapping
			},
		}

		// Set JSON response headers
		w.Header().Set("Content-Type", "application/json")

		// Return JSON response
		json.NewEncoder(w).Encode(response)
	})

	// Endpoint for actual file download
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		// Extract filename from URL path
		filename := strings.TrimPrefix(r.URL.Path, "/download/")
		if filename == "" {
			http.Error(w, "Filename not specified", http.StatusBadRequest)
			return
		}

		// Validate filename to prevent directory traversal
		if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
			http.Error(w, "Invalid filename", http.StatusBadRequest)
			return
		}

		// Look up the file in the config
		mapping := findFileMappingByFilename(config, filename)
		if mapping == nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Serve file content using helper function
		if err := serveFileContent(w, r, *mapping, filename); err != nil {
			http.Error(w, "File not found or error serving file", http.StatusNotFound)
			return
		}
	})

	// Also handle the /upload/firmware/ path that's returned in the JSON response
	mux.HandleFunc("/upload/firmware/", func(w http.ResponseWriter, r *http.Request) {
		// Extract filename from URL path
		filename := strings.TrimPrefix(r.URL.Path, "/upload/firmware/")
		if filename == "" {
			http.Error(w, "Filename not specified", http.StatusBadRequest)
			return
		}

		// Validate filename to prevent directory traversal
		if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
			http.Error(w, "Invalid filename", http.StatusBadRequest)
			return
		}

		// Look up the file in the config
		mapping := findFileMappingByFilename(config, filename)
		if mapping == nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Serve file content using helper function
		if err := serveFileContent(w, r, *mapping, filename); err != nil {
			http.Error(w, "File not found or error serving file", http.StatusNotFound)
			return
		}
	})

	// Wrap the handler with tracing middleware if enabled
	var handler http.Handler = mux
	if *enableTracing {
		handler = tracingMiddleware(mux)
		log.Printf("Request tracing enabled")
	}

	addr := fmt.Sprintf("%s:%s", *bindAddr, *port)
	fmt.Printf("Server starting on http://%s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9190" // default port
	}
	return port
}

func generateId(customerType, firmwareType string) string {
	// Generate a UUID-like string based on the inputs
	return fmt.Sprintf("%s-%s-%d",
		customerType,
		firmwareType,
		time.Now().Unix())
}

func generateName(customerType, firmwareType string) string {
	// Generate a name like "Yuntu_m1_s1_OTA_test"
	return fmt.Sprintf("%s_%s_OTA_test", customerType, strings.Replace(firmwareType, "client", "", -1))
}

func loadConfig(filename string) Config {
	var config Config

	// Try to load from file system first (current directory)
	file, err := os.Open(filename)
	if err != nil {
		// If file system config fails, try embedded config
		log.Printf("Config file %s not found in current directory, trying embedded config...", filename)
		embeddedData, err := embeddedFiles.ReadFile(filename)
		if err != nil {
			log.Fatalf("Failed to read config file from file system and embedded resources: %v", err)
		}

		log.Printf("Using embedded config file: %s", filename)
		decoder := json.NewDecoder(strings.NewReader(string(embeddedData)))
		err = decoder.Decode(&config)
		if err != nil {
			log.Fatalf("Failed to parse embedded config file: %v", err)
		}
	} else {
		// Load from current directory
		defer file.Close()
		log.Printf("Using file system config file: %s", filename)
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&config)
		if err != nil {
			log.Fatalf("Failed to parse config file: %v", err)
		}
	}

	return config
}

// Initialize the firmware cache with pre-calculated data
func initializeFirmwareCache(config Config) {
	firmwareCache = make(map[string]*PreCalculatedFirmwareDetail)

	for _, mapping := range config.Mappings {
		// Create a unique key for each mapping
		key := fmt.Sprintf("%s:%s", mapping.CustomerType, mapping.FirmwareType)

		// Check if file exists and calculate MD5
		exists, err := fileExists(mapping)
		md5Hash := ""

		if err == nil && exists {
			md5Hash, err = calculateMD5ForMapping(mapping)
		}

		firmwareCache[key] = &PreCalculatedFirmwareDetail{
			MD5:        md5Hash,
			FileExists: exists,
			Error:      err,
		}

		if err != nil {
			log.Printf("Warning: Error pre-calculating data for %s: %v", key, err)
		}
	}

	log.Printf("Pre-calculated firmware data for %d mappings", len(config.Mappings))
}

func findMapping(config Config, customerType, firmwareType string) *FirmwareMapping {
	for _, mapping := range config.Mappings {
		if mapping.CustomerType == customerType && mapping.FirmwareType == firmwareType {
			return &mapping
		}
	}
	return nil
}

func findFileMappingByFilename(config Config, filename string) *FirmwareMapping {
	for _, mapping := range config.Mappings {
		// Extract filename from the stored FilePath
		mappingFilename := filepath.Base(mapping.FilePath)

		// Direct comparison (most common case)
		if mappingFilename == filename {
			return &mapping
		}

		// Additional check: in case there are subtle differences in path format
		// or if the filename is provided with relative path components
		cleanMappingFilename := filepath.Base(filepath.Clean(mapping.FilePath))
		if cleanMappingFilename == filename {
			return &mapping
		}
	}
	return nil
}

// tracingMiddleware logs all incoming requests
func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log the incoming request
		log.Printf("TRACE: %s %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		// If it's a query request, log the query parameters too
		if r.URL.RawQuery != "" {
			log.Printf("TRACE: Query params: %s", r.URL.RawQuery)
		}

		// Log request headers if needed (optional)
		// for name, values := range r.Header {
		// 	for _, value := range values {
		// 		log.Printf("TRACE: Header %s: %s", name, value)
		// 	}
		// }

		// Create a response writer that captures the status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Serve the request
		next.ServeHTTP(wrapped, r)

		// Log the response
		duration := time.Since(start)
		log.Printf("TRACE: Response %d in %v", wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Helper function to check if a file exists (either embedded or filesystem)
func fileExists(mapping FirmwareMapping) (bool, error) {
	if mapping.Embed {
		// Check file in embedded filesystem
		relativePath := strings.TrimPrefix(mapping.FilePath, "../")
		relativePath = strings.TrimPrefix(relativePath, "./")

		// Try to open the file as existence check for embedded files)
		f, err := embeddedFiles.Open(relativePath)
		if err != nil {
			return false, err
		}
		f.Close()
		return true, nil
	} else {
		// Check file in filesystem
		_, err := os.Stat(mapping.FilePath)
		if os.IsNotExist(err) {
			return false, nil
		}
		return err == nil, err
	}
}

// Helper function to calculate MD5 for either embedded or filesystem file
func calculateMD5ForMapping(mapping FirmwareMapping) (string, error) {
	var fileReader io.ReadCloser
	var err error

	if mapping.Embed {
		// Open file from embedded filesystem
		relativePath := strings.TrimPrefix(mapping.FilePath, "../")
		relativePath = strings.TrimPrefix(relativePath, "./")
		fileReader, err = embeddedFiles.Open(relativePath)
		if err != nil {
			return "", err
		}
	} else {
		// Open file from filesystem
		fileReader, err = os.Open(mapping.FilePath)
		if err != nil {
			return "", err
		}
	}
	defer fileReader.Close()

	// Calculate MD5 using streaming (same logic for both embedded and filesystem files)
	hash := md5.New()
	if _, err := io.Copy(hash, fileReader); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Helper function to serve file content (either embedded or filesystem)
func serveFileContent(w http.ResponseWriter, r *http.Request, mapping FirmwareMapping, filename string) error {
	if mapping.Embed {
		// Use embedded file path (relative to embed root)
		relativePath := strings.TrimPrefix(mapping.FilePath, "../")
		relativePath = strings.TrimPrefix(relativePath, "./")

		// Open file from embedded filesystem
		embeddedFile, err := embeddedFiles.Open(relativePath)
		if err != nil {
			return err
		}
		defer embeddedFile.Close()

		// Get file info for content length
		fileInfo, err := embeddedFile.Stat()
		if err != nil {
			return err
		}

		// Set appropriate headers for deb file download
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)

		// Since embedded file may not implement io.ReadSeeker, we use io.Copy for streaming
		// Set content length header
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

		// Stream the file content directly to the response
		_, err = io.Copy(w, embeddedFile)
		if err != nil {
			return err
		}
	} else {
		// Check if file exists in filesystem
		if _, err := os.Stat(mapping.FilePath); os.IsNotExist(err) {
			return err
		}

		// Set appropriate headers for deb file download
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)

		// Serve the file from filesystem
		http.ServeFile(w, r, mapping.FilePath)
	}

	return nil
}
