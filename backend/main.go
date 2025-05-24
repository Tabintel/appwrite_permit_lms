package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/appwrite/sdk-for-go"
	"github.com/appwrite/sdk-for-go/database"
	"github.com/appwrite/sdk-for-go/query"
	"github.com/appwrite/sdk-for-go/users"
	"github.com/gorilla/mux"
	"github.com/permitio/permit-golang/pkg/permit"
)

// Configuration
type Config struct {
	AppwriteEndpoint string
	AppwriteProject  string
	AppwriteAPIKey   string
	PermitToken      string
	PermitEnv        string
}

// Models
type Course struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	TeacherID   string   `json:"teacherId"`
	StudentIDs  []string `json:"studentIds"`
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

// Service
type LMSService struct {
	appwrite *sdk.Client
	permit   *permit.Permit
	config   Config
}

func NewLMSService(config Config) (*LMSService, error) {
	// Initialize Appwrite client
	client := sdk.NewClient()
	client.SetEndpoint(config.AppwriteEndpoint).
		SetProject(config.AppwriteProject).
		SetKey(config.AppwriteAPIKey)

	// Initialize Permit client
	permitClient, err := permit.NewPermit(
		permit.WithToken(config.PermitToken),
		permit.WithEnvironment(config.PermitEnv),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Permit client: %w", err)
	}

	return &LMSService{
		appwrite: client,
		permit:   permitClient,
		config:   config,
	}, nil
}

// Middleware for authentication and authorization
func (s *LMSService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get JWT token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		token := tokenParts[1]

		// Verify token with Appwrite (in a real app)
		// For demo purposes, we'll assume the token is valid and contains user info
		// In a real app, you would verify the token with Appwrite and extract user info

		// Mock user info for demo
		userID := "user123"
		userRole := "student"

		// Store user info in request context
		ctx := context.WithValue(r.Context(), "userID", userID)
		ctx = context.WithValue(ctx, "userRole", userRole)

		// Continue with the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Course handlers
func (s *LMSService) GetCourses(w http.ResponseWriter, r *http.Request) {
	// Get user info from context
	userID := r.Context().Value("userID").(string)
	userRole := r.Context().Value("userRole").(string)

	// Initialize database client
	db := database.NewClient(s.appwrite)

	var courses []Course
	var err error

	// Get courses based on user role
	switch userRole {
	case "admin":
		// Admins can see all courses
		result, err := db.ListDocuments(
			context.Background(),
			"courses",
			[]string{}, // No queries, get all
			100,        // Limit
			0,          // Offset
			"",         // Cursor
			[]string{}, // Orders
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get courses: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Parse courses from result
		if err := json.Unmarshal([]byte(result.String()), &courses); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse courses: %v", err), http.StatusInternalServerError)
			return
		}

	case "teacher":
		// Teachers can see courses they teach
		result, err := db.ListDocuments(
			context.Background(),
			"courses",
			[]string{query.Equal("teacherId", userID)},
			100, // Limit
			0,   // Offset
			"",  // Cursor
			[]string{}, // Orders
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get courses: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Parse courses from result
		if err := json.Unmarshal([]byte(result.String()), &courses); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse courses: %v", err), http.StatusInternalServerError)
			return
		}

	case "student":
		// Students can see courses they're enrolled in
		// This is where fine-grained authorization with Permit comes in
		
		// First, get all courses
		result, err := db.ListDocuments(
			context.Background(),
			"courses",
			[]string{}, // No queries, get all
			100,        // Limit
			0,          // Offset
			"",         // Cursor
			[]string{}, // Orders
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get courses: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Parse all courses
		var allCourses []Course
		if err := json.Unmarshal([]byte(result.String()), &allCourses); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse courses: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Filter courses using Permit.io
		for _, course := range allCourses {
			// Check if student can access this course using Permit
			allowed, err := s.permit.Check(
				context.Background(),
				userID,                // User ID
				"read",                // Action
				"course:"+course.ID,   // Resource
			)
			if err != nil {
				log.Printf("Permit check error: %v", err)
				continue
			}
			
			if allowed {
				courses = append(courses, course)
			}
		}
	}

	// Return courses as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(courses)
}

func (s *LMSService) CreateCourse(w http.ResponseWriter, r *http.Request) {
	// Get user info from context
	userID := r.Context().Value("userID").(string)
	userRole := r.Context().Value("userRole").(string)

	// Check if user can create a course using Permit
	allowed, err := s.permit.Check(
		context.Background(),
		userID,     // User ID
		"create",   // Action
		"course",   // Resource
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check permissions: %v", err), http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Parse course data from request
	var course Course
	if err := json.NewDecoder(r.Body).Decode(&course); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse course data: %v", err), http.StatusBadRequest)
		return
	}

	// Set teacher ID based on user role
	if userRole == "teacher" {
		course.TeacherID = userID
	}

	// Initialize database client
	db := database.NewClient(s.appwrite)

	// Create course in Appwrite
	result, err := db.CreateDocument(
		context.Background(),
		"courses",
		"unique()",
		map[string]interface{}{
			"title":       course.Title,
			"description": course.Description,
			"teacherId":   course.TeacherID,
			"studentIds":  course.StudentIDs,
		},
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create course: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse created course
	var createdCourse Course
	if err := json.Unmarshal([]byte(result.String()), &createdCourse); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse created course: %v", err), http.StatusInternalServerError)
		return
	}

	// Sync the new course with Permit.io
	// This would create a resource instance in Permit
	// In a real app, you would implement this

	// Return created course as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdCourse)
}

func (s *LMSService) EnrollInCourse(w http.ResponseWriter, r *http.Request) {
	// Get user info from context
	userID := r.Context().Value("userID").(string)
	userRole := r.Context().Value("userRole").(string)

	// Only students can enroll in courses
	if userRole != "student" {
		http.Error(w, "Only students can enroll in courses", http.StatusForbidden)
		return
	}

	// Get course ID from URL
	vars := mux.Vars(r)
	courseID := vars["courseID"]

	// Check if user can enroll in this course using Permit
	allowed, err := s.permit.Check(
		context.Background(),
		userID,                // User ID
		"enroll",              // Action
		"course:"+courseID,    // Resource
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check permissions: %v", err), http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Initialize database client
	db := database.NewClient(s.appwrite)

	// Get course from Appwrite
	result, err := db.GetDocument(
		context.Background(),
		"courses",
		courseID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get course: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse course
	var course Course
	if err := json.Unmarshal([]byte(result.String()), &course); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse course: %v", err), http.StatusInternalServerError)
		return
	}

	// Add student to course
	course.StudentIDs = append(course.StudentIDs, userID)

	// Update course in Appwrite
	result, err = db.UpdateDocument(
		context.Background(),
		"courses",
		courseID,
		map[string]interface{}{
			"studentIds": course.StudentIDs,
		},
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update course: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse updated course
	var updatedCourse Course
	if err := json.Unmarshal([]byte(result.String()), &updatedCourse); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse updated course: %v", err), http.StatusInternalServerError)
		return
	}

	// Sync the enrollment with Permit.io
	// This would update the relationship between the user and the course in Permit
	// In a real app, you would implement this

	// Return success message
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Successfully enrolled in course",
	})
}

// Main function
func main() {
	// Load configuration from environment variables
	config := Config{
		AppwriteEndpoint: os.Getenv("APPWRITE_ENDPOINT"),
		AppwriteProject:  os.Getenv("APPWRITE_PROJECT"),
		AppwriteAPIKey:   os.Getenv("APPWRITE_API_KEY"),
		PermitToken:      os.Getenv("PERMIT_TOKEN"),
		PermitEnv:        os.Getenv("PERMIT_ENV"),
	}

	// Create LMS service
	service, err := NewLMSService(config)
	if err != nil {
		log.Fatalf("Failed to create LMS service: %v", err)
	}

	// Create router
	r := mux.NewRouter()

	// Apply middleware
	r.Use(service.AuthMiddleware)

	// Define routes
	r.HandleFunc("/courses", service.GetCourses).Methods("GET")
	r.HandleFunc("/courses", service.CreateCourse).Methods("POST")
	r.HandleFunc("/courses/{courseID}/enroll", service.EnrollInCourse).Methods("POST")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
