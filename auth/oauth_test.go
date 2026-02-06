package auth

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultScopes(t *testing.T) {
	scopes := DefaultScopes()

	if len(scopes) == 0 {
		t.Error("DefaultScopes returned empty slice")
	}

	expectedScopes := []string{
		"https://www.googleapis.com/auth/calendar",
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/gmail.modify",
		"https://www.googleapis.com/auth/spreadsheets",
		"https://www.googleapis.com/auth/documents",
		"https://www.googleapis.com/auth/presentations",
		"https://www.googleapis.com/auth/tasks",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}

	if len(scopes) != len(expectedScopes) {
		t.Errorf("Expected %d scopes, got %d", len(expectedScopes), len(scopes))
	}

	// Check each scope
	scopeMap := make(map[string]bool)
	for _, scope := range scopes {
		scopeMap[scope] = true
	}

	for _, expected := range expectedScopes {
		if !scopeMap[expected] {
			t.Errorf("Missing expected scope: %s", expected)
		}
	}
}

func TestOAuthConfig(t *testing.T) {
	config := OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
		TokenFile:    filepath.Join(os.TempDir(), "test-token.json"),
		Scopes:       DefaultScopes(),
	}

	if config.ClientID != "test-client-id" {
		t.Errorf("Expected ClientID to be 'test-client-id', got %s", config.ClientID)
	}

	if config.ClientSecret != "test-client-secret" {
		t.Errorf("Expected ClientSecret to be 'test-client-secret', got %s", config.ClientSecret)
	}
}

func TestNewOAuthClientWithoutAuth(t *testing.T) {
	// Skip this test on Windows or CI environments to avoid OAuth flow
	if os.Getenv("CI") != "" {
		t.Skip("Skipping OAuth test in CI environment")
	}

	ctx := context.Background()

	// Test with missing credentials (should fail gracefully)
	config := OAuthConfig{
		ClientID:     "",
		ClientSecret: "",
	}

	_, err := NewOAuthClient(ctx, config)
	if err == nil {
		t.Error("Expected error with empty credentials")
	}
}

func TestGenerateOAuthState(t *testing.T) {
	state1, err := generateOAuthState()
	if err != nil {
		t.Fatalf("generateOAuthState() returned error: %v", err)
	}

	// Should be 64 hex characters (32 bytes encoded)
	if len(state1) != 64 {
		t.Errorf("Expected state length 64, got %d", len(state1))
	}

	// Should be valid hex
	if _, err := hex.DecodeString(state1); err != nil {
		t.Errorf("State is not valid hex: %v", err)
	}

	// Two consecutive calls should produce different states
	state2, err := generateOAuthState()
	if err != nil {
		t.Fatalf("second generateOAuthState() returned error: %v", err)
	}
	if state1 == state2 {
		t.Error("Two consecutive calls produced the same state")
	}
}

func TestTokenFilePath(t *testing.T) {
	tempDir := t.TempDir()
	tokenFile := filepath.Join(tempDir, "test-token.json")

	config := OAuthConfig{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		TokenFile:    tokenFile,
	}

	// Verify the path is correctly set
	if config.TokenFile != tokenFile {
		t.Errorf("Expected token file path %s, got %s", tokenFile, config.TokenFile)
	}

	// Verify the directory exists or can be created
	dir := filepath.Dir(config.TokenFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("Token file directory does not exist: %s", dir)
	}
}
