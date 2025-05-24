package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/appwrite/sdk-for-go/v10"
	"github.com/appwrite/sdk-for-go/v10/database"
	"github.com/permitio/permit-sdk-go/pkg/permit"
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
		UserID    string `json:"userId"`
		UserRole  string `json:"userRole"`
		CourseID  string `json:"courseId"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		respondWithError("Failed to parse request", err)
		return
	}

	// Only students can enroll in courses
	if req.UserRole != "student" {
		respondWithError("Only students can enroll in courses", fmt.Errorf("user role is %s", req.UserRole))
		return
	}

	// Check if user can enroll in this course using Permit
	allowed, err := permitClient.Check(
		context.Background(),
		req.UserID,                // User ID
		"enroll",                  // Action
		"course:"+req.CourseID,    // Resource
	)
	if err != nil {
		respondWithError("Failed to check permissions", err)
		return
	}

	if !allowed {
		respondWithError("Permission denied", fmt.Errorf("user does not have permission to enroll in this course"))
		return
	}

	// Initialize database client
	db := database.NewClient(client)

	// Get course from Appwrite
	result, err := db.GetDocument(
		context.Background(),
		"courses",
		req.CourseID,
	)
	if err != nil {
		respondWithError("Failed to get course", err)
		return
	}

	// Parse course
	var course Course
	if err := json.Unmarshal([]byte(result.String()), &course); err != nil {
		respondWithError("Failed to parse course", err)
		return
	}

	// Check if student is already enrolled
	for _, studentID := range course.StudentIDs {
		if studentID == req.UserID {
			respondWithError("Student already enrolled", fmt.Errorf("student is already enrolled in this course"))
			return
		}
	}

	// Add student to course
	course.StudentIDs = append(course.StudentIDs, req.UserID)

	// Update course in Appwrite
	result, err = db.UpdateDocument(
		context.Background(),
		"courses",
		req.CourseID,
		map[string]interface{}{
			"studentIds": course.StudentIDs,
		},
	)
	if err != nil {
		respondWithError("Failed to update course", err)
		return
	}

	// Parse updated course
	var updatedCourse Course
	if err := json.Unmarshal([]byte(result.String()), &updatedCourse); err != nil {
		respondWithError("Failed to parse updated course", err)
		return
	}

	// Sync the enrollment with Permit.io
	_, err = permitClient.SyncResource(context.Background(), "course", req.CourseID, map[string]interface{}{
		"teacherId":  updatedCourse.TeacherID,
		"studentIds": updatedCourse.StudentIDs,
	})
	if err != nil {
		log.Printf("Failed to sync course %s: %v", req.CourseID, err)
	}

	// Return success message
	respondWithSuccess("Successfully enrolled in course", updatedCourse)
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
