package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/appwrite/sdk-for-go"
	"github.com/appwrite/sdk-for-go/database"
	"github.com/appwrite/sdk-for-go/query"
	"github.com/permitio/permit-golang/pkg/permit"
)

// Course represents a course in the LMS
type Course struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	TeacherID   string   `json:"teacherId"`
	StudentIDs  []string `json:"studentIds"`
}

// Response is the standard response format for Appwrite functions
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	// Initialize Appwrite client
	client := sdk.NewClient()
	client.SetEndpoint(os.Getenv("APPWRITE_ENDPOINT"))
	client.SetProject(os.Getenv("APPWRITE_FUNCTION_PROJECT_ID"))
	client.SetKey(os.Getenv("APPWRITE_API_KEY"))

	// Initialize Permit client
	permitClient, err := permit.NewPermit(
		permit.WithToken(os.Getenv("PERMIT_TOKEN")),
		permit.WithEnvironment(os.Getenv("PERMIT_ENV")),
		permit.WithPDP(permit.PDPConfig{
			Address: os.Getenv("PERMIT_PDP_ADDRESS"),
		}),
	)
	if err != nil {
		log.Fatalf("Failed to initialize Permit client: %v", err)
	}

	// Parse request
	var req struct {
		UserID   string `json:"userId"`
		UserRole string `json:"userRole"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		respondWithError("Failed to parse request", err)
		return
	}

	// Get courses based on user role
	courses, err := getCourses(client, permitClient, req.UserID, req.UserRole)
	if err != nil {
		respondWithError("Failed to get courses", err)
		return
	}

	// Return courses
	respondWithSuccess("Courses retrieved successfully", courses)
}

func getCourses(client *sdk.Client, permitClient *permit.Permit, userID, userRole string) ([]Course, error) {
	// Initialize database client
	db := database.NewClient(client)
	var courses []Course

	switch userRole {
	case "admin":
		// Admins can see all courses
		result, err := db.ListDocuments(
			context.Background(),
			"courses",
			[]interface{}{}, // No queries, get all
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get courses: %w", err)
		}

		// Parse courses from result
		if err := json.Unmarshal([]byte(result.String()), &courses); err != nil {
			return nil, fmt.Errorf("failed to parse courses: %w", err)
		}

	case "teacher":
		// Teachers can see courses they teach
		result, err := db.ListDocuments(
			context.Background(),
			"courses",
			[]interface{}{query.Equal("teacherId", userID)},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get courses: %w", err)
		}

		// Parse courses from result
		if err := json.Unmarshal([]byte(result.String()), &courses); err != nil {
			return nil, fmt.Errorf("failed to parse courses: %w", err)
		}

	case "student":
		// Students can see courses they're enrolled in
		// First, get all courses
		result, err := db.ListDocuments(
			context.Background(),
			"courses",
			[]interface{}{}, // No queries, get all
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get courses: %w", err)
		}

		// Parse all courses
		var allCourses []Course
		if err := json.Unmarshal([]byte(result.String()), &allCourses); err != nil {
			return nil, fmt.Errorf("failed to parse courses: %w", err)
		}

		// Filter courses using Permit.io
		for _, course := range allCourses {
			// Check if student can access this course using Permit
			allowed, err := permitClient.Check(
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

	return courses, nil
}

func respondWithSuccess(message string, data interface{}) {
	response := Response{
		Success: true,
		Message: message,
		Data:    data,
	}
	json.NewEncoder(os.Stdout).Encode(response)
}

func respondWithError(message string, err error) {
	response := Response{
		Success: false,
		Message: fmt.Sprintf("%s: %v", message, err),
	}
	json.NewEncoder(os.Stdout).Encode(response)
}
