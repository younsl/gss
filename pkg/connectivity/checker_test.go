package connectivity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewChecker(t *testing.T) {
	// Test with default values
	config := Config{
		BaseURL: "https://github.example.com",
	}
	checker := NewChecker(config)

	if checker.config.MaxRetries != 3 {
		t.Errorf("Expected default MaxRetries to be 3, got %d", checker.config.MaxRetries)
	}
	if checker.config.RetryInterval != 5 {
		t.Errorf("Expected default RetryInterval to be 5, got %d", checker.config.RetryInterval)
	}
	if checker.config.Timeout != 5 {
		t.Errorf("Expected default Timeout to be 5, got %d", checker.config.Timeout)
	}

	// Test with custom values
	customConfig := Config{
		BaseURL:       "https://github.example.com",
		MaxRetries:    5,
		RetryInterval: 2,
		Timeout:       15,
	}
	customChecker := NewChecker(customConfig)

	if customChecker.config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", customChecker.config.MaxRetries)
	}
	if customChecker.config.RetryInterval != 2 {
		t.Errorf("Expected RetryInterval to be 2, got %d", customChecker.config.RetryInterval)
	}
	if customChecker.config.Timeout != 15 {
		t.Errorf("Expected Timeout to be 15, got %d", customChecker.config.Timeout)
	}
}

func TestVerifyConnectivitySuccess(t *testing.T) {
	// Create mock server info
	mockServerInfo := ServerInfo{
		VerifiablePasswordAuth: false,
		InstalledVersion:       "3.13.9",
	}

	// Create a test server that returns server info
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockServerInfo)
	}))
	defer server.Close()

	config := Config{
		BaseURL:       server.URL,
		MaxRetries:    1,
		RetryInterval: 1,
		Timeout:       1,
	}
	checker := NewChecker(config)

	// Test connectivity
	err := checker.VerifyConnectivity()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestVerifyConnectivityFailure(t *testing.T) {
	// Create a test server that returns a 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := Config{
		BaseURL:       server.URL,
		MaxRetries:    2,
		RetryInterval: 1,
		Timeout:       1,
	}
	checker := NewChecker(config)

	// Test connectivity
	err := checker.VerifyConnectivity()
	if err == nil {
		t.Error("Expected an error, got nil")
	}
}

func TestVerifyConnectivityInvalidURL(t *testing.T) {
	config := Config{
		BaseURL:       "://invalid-url",
		MaxRetries:    1,
		RetryInterval: 1,
		Timeout:       1,
	}
	checker := NewChecker(config)

	// Test connectivity
	err := checker.VerifyConnectivity()
	if err == nil {
		t.Error("Expected an error for invalid URL, got nil")
	}
}

func TestVerifyConnectivityTimeout(t *testing.T) {
	// Create a test server that sleeps longer than the timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		BaseURL:       server.URL,
		MaxRetries:    1,
		RetryInterval: 1,
		Timeout:       1,
	}
	checker := NewChecker(config)

	// Test connectivity
	err := checker.VerifyConnectivity()
	if err == nil {
		t.Error("Expected a timeout error, got nil")
	}
}

func TestGetServerInfo(t *testing.T) {
	// Create mock server info
	mockServerInfo := ServerInfo{
		VerifiablePasswordAuth: false,
		InstalledVersion:       "3.13.9",
	}

	// Create a test server that returns server info
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockServerInfo)
	}))
	defer server.Close()

	config := Config{
		BaseURL:       server.URL,
		MaxRetries:    1,
		RetryInterval: 1,
		Timeout:       1,
	}
	checker := NewChecker(config)

	// Test getting server info
	info, err := checker.GetServerInfo()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if info == nil {
		t.Fatal("Expected server info, got nil")
	}

	if info.InstalledVersion != mockServerInfo.InstalledVersion {
		t.Errorf("Expected installed version %s, got %s", mockServerInfo.InstalledVersion, info.InstalledVersion)
	}

	if info.VerifiablePasswordAuth != mockServerInfo.VerifiablePasswordAuth {
		t.Errorf("Expected verifiable password auth %v, got %v", mockServerInfo.VerifiablePasswordAuth, info.VerifiablePasswordAuth)
	}
}

func TestGetServerInfoFailure(t *testing.T) {
	// Create a test server that returns a 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := Config{
		BaseURL:       server.URL,
		MaxRetries:    1,
		RetryInterval: 1,
		Timeout:       1,
	}
	checker := NewChecker(config)

	// Test getting server info
	info, err := checker.GetServerInfo()
	if err == nil {
		t.Error("Expected an error, got nil")
	}

	if info != nil {
		t.Errorf("Expected nil server info, got %+v", info)
	}
}

func TestGetServerInfoInvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid json}"))
	}))
	defer server.Close()

	config := Config{
		BaseURL:       server.URL,
		MaxRetries:    1,
		RetryInterval: 1,
		Timeout:       1,
	}
	checker := NewChecker(config)

	// Test getting server info
	info, err := checker.GetServerInfo()
	if err == nil {
		t.Error("Expected an error for invalid JSON, got nil")
	}

	if info != nil {
		t.Errorf("Expected nil server info, got %+v", info)
	}
}

func TestMustVerifyConnectivity(t *testing.T) {
	// Create mock server info
	mockServerInfo := ServerInfo{
		VerifiablePasswordAuth: false,
		InstalledVersion:       "3.13.9",
	}

	// Create a test server that returns server info
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockServerInfo)
	}))
	defer server.Close()

	config := Config{
		BaseURL:       server.URL,
		MaxRetries:    1,
		RetryInterval: 1,
		Timeout:       1,
	}
	checker := NewChecker(config)

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustVerifyConnectivity panicked unexpectedly: %v", r)
		}
	}()

	checker.MustVerifyConnectivity()
}

func TestMustVerifyConnectivityPanic(t *testing.T) {
	// Create a test server that returns a 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := Config{
		BaseURL:       server.URL,
		MaxRetries:    1,
		RetryInterval: 1,
		Timeout:       1,
	}
	checker := NewChecker(config)

	// This should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustVerifyConnectivity did not panic as expected")
		}
	}()

	checker.MustVerifyConnectivity()
}
