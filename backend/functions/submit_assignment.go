package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/appwrite/sdk-for-go"
	"github.com/appwrite/sdk-for-go/database"
	"github.com/permitio/permit-golang/pkg/permit"
)

// Assignment represents an assignment in the LMS
type Assignment struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CourseID    string `json:"courseId"`
	DueDate     string `json:"dueDate"`
}

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
		AssignmentID string `json:"assignmentId"`
		Content      string `json:"content"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		respondWithError("Failed to parse request", err)
		return
	}

	// Only students can submit assignments
	if req.UserRole != "student" {
		respondWithError("Only students can submit assignments", fmt.Errorf("user role is %s", req.UserRole))
		return
	}

	// Check if user can submit this assignment using Permit
	allowed, err := permitClient.Check(
		context.Background(),
		req.UserID,                    // User ID
		"submit",                      // Action
		"assignment:"+req.AssignmentID, // Resource
	)
	if err != nil {
		respondWithError("Failed to check permissions", err)
		return
	}

	if !allowed {
		respondWithError("Permission denied", fmt.Errorf("user does not have permission to submit this assignment"))
		return
	}

	// Initialize database client
	db := database.NewClient(client)

	// Get assignment to check due date
	result, err := db.GetDocument(
		context.Background(),
		"assignments",
		req.AssignmentID,
	)
	if err != nil {
		respondWithError("Failed to get assignment", err)
		return
	}

	// Parse assignment
	var assignment Assignment
	if err := json.Unmarshal([]byte(result.String()), &assignment); err != nil {
		respondWithError("Failed to parse assignment", err)
		return
	}

	// Check if assignment is past due date
	dueDate, err := time.Parse("2006-01-02", assignment.DueDate)
	if err != nil {
		respondWithError("Failed to parse due date", err)
		return
	}

	if time.Now().After(dueDate) {
		respondWithError("Assignment is past due date", fmt.Errorf("due date was %s", assignment.DueDate))
		return
	}

	// Create submission
	submission := Submission{
		AssignmentID: req.AssignmentID,
		StudentID:    req.UserID,
		Content:      req.Content,
		SubmittedAt:  time.Now().Format(time.RFC3339),
		Grade:        0,
		Feedback:     "",
	}

	// Create submission in Appwrite
	result, err = db.CreateDocument(
		context.Background(),
		"submissions",
		"unique()",
		map[string]interface{}{
			"assignmentId": submission.AssignmentID,
			"studentId":    submission.StudentID,
			"content":      submission.Content,
			"submittedAt":  submission.SubmittedAt,
			"grade":        submission.Grade,
			"feedback":     submission.Feedback,
		},
	)
	if err != nil {
		respondWithError("Failed to create submission", err)
		return
	}

	// Parse created submission
	var createdSubmission Submission
	if err := json.Unmarshal([]byte(result.String()), &createdSubmission); err != nil {
		respondWithError("Failed to parse created submission", err)
		return
	}

	// Return created submission
	respondWithSuccess("Submission created successfully", createdSubmission)
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
