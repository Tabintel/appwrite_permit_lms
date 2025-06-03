package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/appwrite/sdk-for-go"
	"github.com/appwrite/sdk-for-go/database"
	"github.com/permitio/permit-golang/pkg/permit"
)

// Submission represents a student's submission for an assignment
type Submission struct {
	ID           string `json:"id"`
	AssignmentID string `json:"assignmentId"`
	StudentID    string `json:"studentId"`
	Content      string `json:"content"`
	SubmittedAt  string `json:"submittedAt"`
	Grade        int    `json:"grade"`
	Feedback     string `json:"feedback"`
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
		UserID       string `json:"userId"`
		UserRole     string `json:"userRole"`
		SubmissionID string `json:"submissionId"`
		Grade        int    `json:"grade"`
		Feedback     string `json:"feedback"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		respondWithError("Failed to parse request", err)
		return
	}

	// Only teachers and admins can grade assignments
	if req.UserRole != "teacher" && req.UserRole != "admin" {
		respondWithError("Only teachers and admins can grade assignments", fmt.Errorf("user role is %s", req.UserRole))
		return
	}

	// Initialize database client
	db := database.NewClient(client)

	// Get submission
	result, err := db.GetDocument(
		context.Background(),
		"submissions",
		req.SubmissionID,
	)
	if err != nil {
		respondWithError("Failed to get submission", err)
		return
	}

	// Parse submission
	var submission Submission
	if err := json.Unmarshal([]byte(result.String()), &submission); err != nil {
		respondWithError("Failed to parse submission", err)
		return
	}

	// Check if user can grade this assignment using Permit
	allowed, err := permitClient.Check(
		context.Background(),
		req.UserID,                            // User ID
		"grade",                               // Action
		"assignment:"+submission.AssignmentID, // Resource
	)
	if err != nil {
		respondWithError("Failed to check permissions", err)
		return
	}

	if !allowed {
		respondWithError("Permission denied", fmt.Errorf("user does not have permission to grade this assignment"))
		return
	}

	// Update submission with grade and feedback
	result, err = db.UpdateDocument(
		context.Background(),
		"submissions",
		req.SubmissionID,
		map[string]interface{}{
			"grade":    req.Grade,
			"feedback": req.Feedback,
		},
	)
	if err != nil {
		respondWithError("Failed to update submission", err)
		return
	}

	// Parse updated submission
	var updatedSubmission Submission
	if err := json.Unmarshal([]byte(result.String()), &updatedSubmission); err != nil {
		respondWithError("Failed to parse updated submission", err)
		return
	}

	// Return updated submission
	respondWithSuccess("Submission graded successfully", updatedSubmission)
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
