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
		UserID      string `json:"userId"`
		UserRole    string `json:"userRole"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		respondWithError("Failed to parse request", err)
		return
	}

	// Check if user can create a course using Permit
	allowed, err := permitClient.Check(
		context.Background(),
		req.UserID,  // User ID
		"create",    // Action
		"course",    // Resource
	)
	if err != nil {
		respondWithError("Failed to check permissions", err)
		return
	}

	if !allowed {
		respondWithError("Permission denied", fmt.Errorf("user does not have permission to create courses"))
		return
	}

	// Create course
	course := Course{
		Title:       req.Title,
		Description: req.Description,
		TeacherID:   req.UserID,
		StudentIDs:  []string{},
	}

	// Initialize database client
	db := database.NewClient(client)

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
		respondWithError("Failed to create course", err)
		return
	}

	// Parse created course
	var createdCourse Course
	if err := json.Unmarshal([]byte(result.String()), &createdCourse); err != nil {
		respondWithError("Failed to parse created course", err)
		return
	}

	// Sync the new course with Permit.io
	_, err = permitClient.SyncResource(context.Background(), "course", createdCourse.ID, map[string]interface{}{
		"teacherId":  createdCourse.TeacherID,
		"studentIds": createdCourse.StudentIDs,
	})
	if err != nil {
		log.Printf("Failed to sync course %s: %v", createdCourse.ID, err)
	}

	// Return created course
	respondWithSuccess("Course created successfully", createdCourse)
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
