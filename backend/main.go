package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/appwrite/go-sdk/appwrite"
	"github.com/appwrite/go-sdk/appwrite/databases"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/permitio/permit-golang/pkg/permit"
	permitConfig "github.com/permitio/permit-golang/pkg/permit/config"
	"github.com/permitio/permit-golang/pkg/permit/models"
)

// Configuration
type Config struct {
	AppwriteEndpoint string `json:"appwrite_endpoint"`
	AppwriteProject  string `json:"appwrite_project"`
	AppwriteAPIKey   string `json:"appwrite_api_key"`
	PermitToken      string `json:"permit_token"`
	PermitPDP        string `json:"permit_pdp"`
	PermitAPIURL     string `json:"permit_api_url"`
}

// Models
type Course struct {
	ID          string        `json:"$id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	TeacherID   string        `json:"teacherId"`
	StudentIDs  []string      `json:"studentIds"`
	Collection  string        `json:"$collection"`
	Permissions []interface{} `json:"$permissions"`
	CreatedAt   string        `json:"$createdAt"`
	UpdatedAt   string        `json:"$updatedAt"`
}

type Assignment struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CourseID    string `json:"courseId"`
	DueDate     string `json:"dueDate"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Service handles the core business logic of the LMS
type LMSService struct {
	client      *appwrite.Client
	db          *databases.Service
	permit      *permit.Client
	config      Config
	databaseID  string
	collectionID string
}

func NewLMSService(config Config) (*LMSService, error) {
	// Initialize Appwrite client
	client := appwrite.NewClient()
	
	// Configure client
	client.SetEndpoint(config.AppwriteEndpoint)
	client.SetProject(config.AppwriteProject)
	client.SetKey(config.AppwriteAPIKey)

	// Initialize Database client
	dbClient := databases.New(client)

	// Initialize Permit client
	permitCfg := permitConfig.NewConfigBuilder(config.PermitToken).
		WithApiUrl(config.PermitAPIURL).
		WithPdpUrl(config.PermitPDP).
		WithDebug(true).
		Build()

	permitClient, err := permit.New(permitCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Permit client: %w", err)
	}

	// Get database and collection IDs from environment or use defaults
	databaseID := os.Getenv("APPWRITE_DATABASE_ID")
	if databaseID == "" {
		databaseID = "default"
	}

	collectionID := os.Getenv("APPWRITE_COLLECTION_ID")
	if collectionID == "" {
		collectionID = "courses"
	}

	return &LMSService{
		client:       client,
		db:           dbClient,
		permit:       permitClient,
		config:       config,
		databaseID:   databaseID,
		collectionID: collectionID,
	}, nil
}

// Middleware for authentication and authorization
func (s *LMSService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for OPTIONS preflight requests
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Get JWT token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		// Expecting format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Initialize Appwrite Account service
		account := appwrite.NewAccount(s.client)

		// Verify the JWT token with Appwrite
		session, err := account.GetSession(token)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Get user details
		user, err := account.Get()
		if err != nil {
			http.Error(w, "Failed to fetch user details", http.StatusInternalServerError)
			return
		}

		// Get user preferences (where we store roles)
		prefs, err := account.GetPrefs()
		userRoles := []string{"user"} // Default role

		// Extract roles from user preferences if available
		if err == nil && prefs != nil {
			if roles, ok := prefs["roles"].([]interface{}); ok {
				userRoles = make([]string, 0, len(roles))
				for _, r := range roles {
					if role, ok := r.(string); ok {
						userRoles = append(userRoles, role)
					}
				}
			}
		}

		// Create user info context
		userInfo := map[string]interface{}{
			"id":      user.Get("$id"),
			"email":   user.Get("email"),
			"name":    user.Get("name"),
			"roles":   userRoles,
			"session": session,
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), "user", userInfo)

		// Add user ID and roles to request headers for downstream services
		r.Header.Set("X-User-ID", user.Get("$id"))
		r.Header.Set("X-User-Email", user.Get("email"))
		r.Header.Set("X-User-Name", user.Get("name"))
		r.Header.Set("X-User-Roles", strings.Join(userRoles, ","))

		// Continue with the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCourses returns a list of courses based on user permissions
func (s *LMSService) GetCourses(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by AuthMiddleware)
	user, ok := getContextUser(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	userID, _ := user["id"].(string)
	userRoles, _ := user["roles"].([]string)

	// Get collection ID from environment or use default
	collectionID := getEnv("APPWRITE_COLLECTION_ID", "courses")

	// Get all courses from Appwrite
	documents, err := s.db.ListDocuments(
		r.Context(),
		s.databaseID,
		collectionID,
	)
	if err != nil {
		log.Printf("Failed to get courses: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve courses")
		return
	}

	// Convert documents to Course slice
	var allCourses []Course
	if err := json.Unmarshal([]byte(documents.(string)), &allCourses); err != nil {
		log.Printf("Failed to unmarshal courses: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to process courses")
		return
	}

	// Filter courses based on permissions
	var filteredCourses []Course
	for _, course := range allCourses {
		// Check permission with Permit.io for each course
		allowed, err := s.permit.Check(
			r.Context(),
			userID,
			"view",
			"course",
			map[string]string{
				"id":        course.ID,
				"teacherId": course.TeacherID,
			},
		)

		if err != nil {
			log.Printf("Permission check failed for course %s: %v", course.ID, err)
			continue
		}

		if allowed {
			filteredCourses = append(filteredCourses, course)
		}
	}

	// Log access for auditing
	log.Printf("User %s with roles %v accessed %d courses", 
		userID, userRoles, len(filteredCourses))

	// Return filtered courses
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    filteredCourses,
		"meta": map[string]interface{}{
			"total":   len(filteredCourses),
			"filtered": len(filteredCourses) < len(allCourses),
		},
	})
}

// CreateCourse handles course creation
func (s *LMSService) CreateCourse(w http.ResponseWriter, r *http.Request) {
	// Get user info from context
	user, ok := getContextUser(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	userID, _ := user["id"].(string)
	userRole, _ := user["role"].(string)

	// Parse request body
	var courseData struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		TeacherID   string `json:"teacherId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&courseData); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Set teacher ID to current user if not provided or not admin
	if courseData.TeacherID == "" || userRole != "admin" {
		courseData.TeacherID = userID
	}

	// Validate required fields
	if courseData.Title == "" {
		respondWithError(w, http.StatusBadRequest, "Title is required")
		return
	}

	// Check if user can create a course using Permit
	allowed, err := s.permit.Check(
		r.Context(),
		userID,
		"create",
		&models.ResourceInput{
			Type: "course",
		},
	)

	if err != nil {
		log.Printf("Error checking permission: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to check permissions")
		return
	}

	if !allowed {
		respondWithError(w, http.StatusForbidden, "Not authorized to create courses")
		return
	}

	// Set teacher ID if not provided (default to current user)
	if newCourse.TeacherID == "" {
		newCourse.TeacherID = userID
	}

	// Check if setting teacher ID for another user (admin only)
	if newCourse.TeacherID != userID {
		isAdmin := false
		for _, role := range user["roles"].([]string) {
			if role == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			respondWithError(w, http.StatusForbidden, "Only admins can create courses for other teachers")
			return
		}
	}

	// Create course document in Appwrite
	doc, err := s.db.CreateDocument(
		r.Context(),
		s.databaseID,
		s.collectionID,
		"unique()", // Let Appwrite generate a unique ID
		map[string]interface{}{
			"title":       courseData.Title,
			"description": courseData.Description,
			"teacherId":   courseData.TeacherID,
			"studentIds":  []string{},
		},
	)

	if err != nil {
		log.Printf("Error creating course: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create course")
		return
	}

	// Sync with Permit.io for fine-grained access control
	_, err = s.permit.Api.SyncResource(context.Background(), &models.ResourceInput{
		Type: "course",
		Key:  doc.Get("$id").(string),
		Attributes: map[string]interface{}{
			"teacherId":   courseData.TeacherID,
			"title":       courseData.Title,
			"description": courseData.Description,
		},
	})

	if err != nil {
		log.Printf("Warning: Failed to sync course with Permit.io: %v", err)
		// Continue even if sync fails, as the course was created successfully
	}

	// Log the course creation
	log.Printf("User %s created course %s", userID, doc.Get("$id"))

	// Return created course with 201 status
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    doc,
	})
}

func (s *LMSService) EnrollInCourse(w http.ResponseWriter, r *http.Request) {
	// ...

	// Parse request body to get course ID
	var requestData struct {
		CourseID string `json:"courseId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Printf("Failed to parse request data: %v", err)
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	courseID := requestData.CourseID
	if courseID == "" {
		http.Error(w, "Course ID is required", http.StatusBadRequest)
		return
	}

	// Check if user can enroll in courses
	if userRole != "student" {
		http.Error(w, "Only students can enroll in courses", http.StatusForbidden)
		return
	}

	// Get the course
	collectionID := os.Getenv("APPWRITE_COLLECTION_ID")
	if collectionID == "" {
		collectionID = "courses"
	}

	// Get document by ID
	doc, err := s.db.GetDocument(
		s.databaseID,
		collectionID,
		courseID,
	)
	if err != nil {
		log.Printf("Course not found: %v", err)
		http.Error(w, "Course not found", http.StatusNotFound)
		return
	}

	// Parse the document into Course struct
	var course Course
	if err := json.Unmarshal([]byte(doc.(string)), &course); err != nil {
		log.Printf("Failed to parse course: %v", err)
		http.Error(w, "Failed to process course", http.StatusInternalServerError)
		return
	}

	// Check if student is already enrolled
	for _, studentID := range course.StudentIDs {
		if studentID == userID {
			http.Error(w, "Already enrolled in this course", http.StatusBadRequest)
			return
		}
	}

	// Add student to course
	course.StudentIDs = append(course.StudentIDs, userID)

	// Update course in Appwrite
	collectionID := os.Getenv("APPWRITE_COLLECTION_ID")
	if collectionID == "" {
		collectionID = "courses"
	}

	// Update document
	_, err = s.db.UpdateDocument(
		s.databaseID,
		collectionID,
		courseID,
		map[string]interface{}{
			"studentIds": course.StudentIDs,
		},
		nil, // permissions
	)
	if err != nil {
		log.Printf("Failed to enroll in course: %v", err)
		http.Error(w, "Failed to enroll in course", http.StatusInternalServerError)
		return
	}

	// In a real app, you would update permissions in Permit.io here
	// to allow the student to access the course resources

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Successfully enrolled in course",
	})
}

// Helper function to get environment variable with default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// respondWithJSON sends a JSON response with status code
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error marshaling response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// respondWithError sends an error response in JSON format
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// getContextUser extracts user information from request context
func getContextUser(r *http.Request) (map[string]interface{}, bool) {
	user, ok := r.Context().Value("user").(map[string]interface{})
	return user, ok
}

// Main function
// This is an Appwrite Function that will be triggered by HTTP requests
func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize configuration
	config := Config{
		AppwriteEndpoint: getEnv("APPWRITE_ENDPOINT", "http://localhost/v1"),
		AppwriteProject:  getEnv("APPWRITE_PROJECT", ""),
		AppwriteAPIKey:   getEnv("APPWRITE_API_KEY", ""),
		PermitToken:      getEnv("PERMIT_TOKEN", ""),
		PermitEnv:        getEnv("PERMIT_ENV", "development"),
	}

	// Initialize services
	service, err := NewLMSService(config)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Set up router
	r := mux.NewRouter()

	// Apply CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Health check endpoint (no auth required)
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.Use(service.AuthMiddleware)

	// Course routes
	api.HandleFunc("/courses", service.GetCourses).Methods("GET")
	api.HandleFunc("/courses", service.CreateCourse).Methods("POST")
	api.HandleFunc("/courses/{id}/enroll", service.EnrollInCourse).Methods("POST")

	// Start server
	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
