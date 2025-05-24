package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/appwrite/sdk-for-go/v10"
	"github.com/appwrite/sdk-for-go/v10/database"
	"github.com/appwrite/sdk-for-go/v10/query"
	"github.com/permitio/permit-sdk-go/pkg/permit"
)

// Assignment represents an assignment in the LMS
type Assignment struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CourseID    string `json:"courseId"`
	DueDate     string `json:"dueDate"`
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

	// Check if user can access this course using Permit
	allowed, err := permitClient.Check(
		context.Background(),
		req.UserID,                // User ID
		"read",                    // Action
		"course:"+req.CourseID,    // Resource
	)
	if err != nil {
		respondWithError("Failed to check permissions", err)
		return
	}

	if !allowed {
		respondWithError("Permission denied", fmt.Errorf("user does not have permission to access this course"))
		return
	}

	// Initialize database client
	db := database.NewClient(client)

	// Get assignments for this course
	result, err := db.ListDocuments(
		context.Background(),
		"assignments",
		[]interface{}{query.Equal("courseId", req.CourseID)},
	)
	if err != nil {
		respondWithError("Failed to get assignments", err)
		return
	}

	// Parse assignments from result
	var assignments []Assignment
	if err := json.Unmarshal([]byte(result.String()), &assignments); err != nil {
		respondWithError("Failed to parse assignments", err)
		return
	}

	// Return assignments
	respondWithSuccess("Assignments retrieved successfully", assignments)
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
