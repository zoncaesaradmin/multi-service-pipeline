package tests

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"testgomodule/internal/client"
)

// TestResult represents the result of a test
type TestResult struct {
	Name        string
	Passed      bool
	Error       error
	Duration    time.Duration
	Description string
}

// TestSuite manages and runs all tests
type TestSuite struct {
	client  *client.Client
	results []TestResult
}

// NewTestSuite creates a new test suite
func NewTestSuite(serviceURL string) *TestSuite {
	return &TestSuite{
		client:  client.NewClient(serviceURL),
		results: make([]TestResult, 0),
	}
}

// RunAllTests runs all integration tests
func (ts *TestSuite) RunAllTests() {
	log.Println("Starting integration tests...")

	// Test health check
	ts.runTest("Health Check", "Verify service is healthy", ts.testHealthCheck)

	// Test user operations
	ts.runTest("Create User", "Create a new user", ts.testCreateUser)
	ts.runTest("Get User", "Retrieve user by ID", ts.testGetUser)
	ts.runTest("Get All Users", "Retrieve all users", ts.testGetAllUsers)
	ts.runTest("Update User", "Update existing user", ts.testUpdateUser)
	ts.runTest("Search Users", "Search users by query", ts.testSearchUsers)
	ts.runTest("Get Stats", "Get service statistics", ts.testGetStats)
	ts.runTest("Delete User", "Delete user by ID", ts.testDeleteUser)

	// Test error scenarios
	ts.runTest("Get Non-existent User", "Test error handling for non-existent user", ts.testGetNonExistentUser)
	ts.runTest("Create Duplicate User", "Test error handling for duplicate user", ts.testCreateDuplicateUser)

	// Performance tests
	ts.runTest("Performance - Multiple Users", "Create and manage multiple users", ts.testPerformanceMultipleUsers)

	ts.printResults()
}

// runTest executes a single test and records the result
func (ts *TestSuite) runTest(name, description string, testFunc func() error) {
	start := time.Now()
	err := testFunc()
	duration := time.Since(start)

	result := TestResult{
		Name:        name,
		Passed:      err == nil,
		Error:       err,
		Duration:    duration,
		Description: description,
	}

	ts.results = append(ts.results, result)

	status := "PASS"
	if err != nil {
		status = "FAIL"
	}

	log.Printf("[%s] %s (%v) - %s", status, name, duration, description)
	if err != nil {
		log.Printf("  Error: %v", err)
	}
}

// testHealthCheck tests the health endpoint
func (ts *TestSuite) testHealthCheck() error {
	health, err := ts.client.HealthCheck()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if health.Status != "healthy" {
		return fmt.Errorf("expected status 'healthy', got '%s'", health.Status)
	}

	return nil
}

// testCreateUser tests user creation
func (ts *TestSuite) testCreateUser() error {
	req := &client.CreateUserRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
	}

	user, err := ts.client.CreateUser(req)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if user.Username != req.Username {
		return fmt.Errorf("expected username '%s', got '%s'", req.Username, user.Username)
	}

	if user.Email != req.Email {
		return fmt.Errorf("expected email '%s', got '%s'", req.Email, user.Email)
	}

	return nil
}

// testGetUser tests retrieving a user by ID
func (ts *TestSuite) testGetUser() error {
	// First create a user
	req := &client.CreateUserRequest{
		Username:  "getuser",
		Email:     "getuser@example.com",
		FirstName: "Get",
		LastName:  "User",
	}

	createdUser, err := ts.client.CreateUser(req)
	if err != nil {
		return fmt.Errorf("failed to create user for get test: %w", err)
	}

	// Now retrieve the user
	user, err := ts.client.GetUser(createdUser.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user.ID != createdUser.ID {
		return fmt.Errorf("expected ID %d, got %d", createdUser.ID, user.ID)
	}

	if user.Username != req.Username {
		return fmt.Errorf("expected username '%s', got '%s'", req.Username, user.Username)
	}

	return nil
}

// testGetAllUsers tests retrieving all users
func (ts *TestSuite) testGetAllUsers() error {
	users, err := ts.client.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get all users: %w", err)
	}

	if len(users) < 2 {
		return fmt.Errorf("expected at least 2 users (from previous tests), got %d", len(users))
	}

	return nil
}

// testUpdateUser tests updating a user
func (ts *TestSuite) testUpdateUser() error {
	// First create a user
	req := &client.CreateUserRequest{
		Username:  "updateuser",
		Email:     "updateuser@example.com",
		FirstName: "Update",
		LastName:  "User",
	}

	createdUser, err := ts.client.CreateUser(req)
	if err != nil {
		return fmt.Errorf("failed to create user for update test: %w", err)
	}

	// Now update the user
	newEmail := "updated@example.com"
	updateReq := &client.UpdateUserRequest{
		Email: &newEmail,
	}

	updatedUser, err := ts.client.UpdateUser(createdUser.ID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if updatedUser.Email != newEmail {
		return fmt.Errorf("expected email '%s', got '%s'", newEmail, updatedUser.Email)
	}

	if updatedUser.Username != req.Username {
		return fmt.Errorf("username should not have changed, expected '%s', got '%s'", req.Username, updatedUser.Username)
	}

	return nil
}

// testSearchUsers tests user search functionality
func (ts *TestSuite) testSearchUsers() error {
	// Search for a user we know exists
	users, err := ts.client.SearchUsers("testuser")
	if err != nil {
		return fmt.Errorf("failed to search users: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("expected to find at least 1 user with username 'testuser', got 0")
	}

	found := false
	for _, user := range users {
		if user.Username == "testuser" {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("expected to find user with username 'testuser' in search results")
	}

	return nil
}

// testGetStats tests retrieving service statistics
func (ts *TestSuite) testGetStats() error {
	stats, err := ts.client.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	totalUsers, exists := stats["total_users"]
	if !exists {
		return fmt.Errorf("expected 'total_users' in stats")
	}

	// Convert to float64 (JSON number type) then to int
	totalUsersFloat, ok := totalUsers.(float64)
	if !ok {
		return fmt.Errorf("expected total_users to be a number, got %T", totalUsers)
	}

	if int(totalUsersFloat) < 3 {
		return fmt.Errorf("expected at least 3 users in stats, got %d", int(totalUsersFloat))
	}

	return nil
}

// testDeleteUser tests deleting a user
func (ts *TestSuite) testDeleteUser() error {
	// First create a user
	req := &client.CreateUserRequest{
		Username:  "deleteuser",
		Email:     "deleteuser@example.com",
		FirstName: "Delete",
		LastName:  "User",
	}

	createdUser, err := ts.client.CreateUser(req)
	if err != nil {
		return fmt.Errorf("failed to create user for delete test: %w", err)
	}

	// Now delete the user
	err = ts.client.DeleteUser(createdUser.ID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Verify user is deleted by trying to get it
	_, err = ts.client.GetUser(createdUser.ID)
	if err == nil {
		return fmt.Errorf("expected error when getting deleted user, but got none")
	}

	return nil
}

// testGetNonExistentUser tests error handling for non-existent user
func (ts *TestSuite) testGetNonExistentUser() error {
	_, err := ts.client.GetUser(99999)
	if err == nil {
		return fmt.Errorf("expected error when getting non-existent user, but got none")
	}

	return nil
}

// testCreateDuplicateUser tests error handling for duplicate user
func (ts *TestSuite) testCreateDuplicateUser() error {
	req := &client.CreateUserRequest{
		Username:  "duplicateuser",
		Email:     "duplicate@example.com",
		FirstName: "Duplicate",
		LastName:  "User",
	}

	// Create first user
	_, err := ts.client.CreateUser(req)
	if err != nil {
		return fmt.Errorf("failed to create first user: %w", err)
	}

	// Try to create duplicate user
	_, err = ts.client.CreateUser(req)
	if err == nil {
		return fmt.Errorf("expected error when creating duplicate user, but got none")
	}

	return nil
}

// testPerformanceMultipleUsers tests performance with multiple users
func (ts *TestSuite) testPerformanceMultipleUsers() error {
	const numUsers = 10
	userIDs := make([]int, 0, numUsers)

	// Create multiple users
	for i := 0; i < numUsers; i++ {
		req := &client.CreateUserRequest{
			Username:  "perfuser" + strconv.Itoa(i),
			Email:     "perfuser" + strconv.Itoa(i) + "@example.com",
			FirstName: "Perf",
			LastName:  "User" + strconv.Itoa(i),
		}

		user, err := ts.client.CreateUser(req)
		if err != nil {
			return fmt.Errorf("failed to create user %d: %w", i, err)
		}
		userIDs = append(userIDs, user.ID)
	}

	// Retrieve all users to test performance
	users, err := ts.client.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get all users: %w", err)
	}

	if len(users) < numUsers {
		return fmt.Errorf("expected at least %d users, got %d", numUsers, len(users))
	}

	// Clean up by deleting the performance test users
	for _, id := range userIDs {
		if err := ts.client.DeleteUser(id); err != nil {
			log.Printf("Warning: failed to clean up user %d: %v", id, err)
		}
	}

	return nil
}

// printResults prints a summary of all test results
func (ts *TestSuite) printResults() {
	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	separator := "============================================================"

	log.Println("\n" + separator)
	log.Println("TEST RESULTS SUMMARY")
	log.Println(separator)

	for _, result := range ts.results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
			failed++
		} else {
			passed++
		}

		log.Printf("%-20s | %-4s | %8v | %s", result.Name, status, result.Duration, result.Description)
		if result.Error != nil {
			log.Printf("                     |      |          | Error: %v", result.Error)
		}

		totalDuration += result.Duration
	}

	log.Println(separator)
	log.Printf("Total Tests: %d | Passed: %d | Failed: %d | Duration: %v",
		len(ts.results), passed, failed, totalDuration)

	if failed > 0 {
		log.Printf("❌ %d test(s) failed", failed)
	} else {
		log.Printf("✅ All tests passed!")
	}
}
