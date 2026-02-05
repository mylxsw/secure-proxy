package otp

import (
	"strings"
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	service := NewService("TestIssuer")
	if service == nil {
		t.Fatal("NewService returned nil")
	}
	if service.issuer != "TestIssuer" {
		t.Errorf("expected issuer 'TestIssuer', got '%s'", service.issuer)
	}
}

func TestGenerateSecret(t *testing.T) {
	service := NewService("TestIssuer")

	secret, err := service.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}

	// Secret should be base32 encoded and 32 characters long (20 bytes)
	if len(secret) != 32 {
		t.Errorf("expected secret length 32, got %d", len(secret))
	}

	// Should only contain valid base32 characters
	for _, c := range secret {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567", c) {
			t.Errorf("invalid character in secret: %c", c)
		}
	}

	// Generate another secret and ensure they are different
	secret2, err := service.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}
	if secret == secret2 {
		t.Error("two generated secrets should be different")
	}
}

func TestGenerateProvisioningURI(t *testing.T) {
	service := NewService("TestApp")
	secret := "JBSWY3DPEHPK3PXP"
	account := "user@example.com"

	uri := service.GenerateProvisioningURI(account, secret)

	// Check URI format
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Errorf("URI should start with 'otpauth://totp/', got: %s", uri)
	}
	if !strings.Contains(uri, "secret="+secret) {
		t.Errorf("URI should contain secret, got: %s", uri)
	}
	if !strings.Contains(uri, "issuer=TestApp") {
		t.Errorf("URI should contain issuer, got: %s", uri)
	}
}

func TestVerify(t *testing.T) {
	service := NewService("TestIssuer")
	secret := "JBSWY3DPEHPK3PXP"

	// Get current code and verify it
	currentCode := service.GetCurrentCode(secret)
	if !service.Verify(secret, currentCode) {
		t.Error("Verify should return true for current code")
	}

	// Test with wrong code
	if service.Verify(secret, "000000") && currentCode != "000000" {
		t.Error("Verify should return false for wrong code")
	}

	// Test with spaces in code (should be trimmed)
	codeWithSpaces := currentCode[:3] + " " + currentCode[3:]
	if !service.Verify(secret, codeWithSpaces) {
		t.Error("Verify should handle codes with spaces")
	}
}

func TestVerifyWithWindow(t *testing.T) {
	service := NewService("TestIssuer")
	secret := "JBSWY3DPEHPK3PXP"

	// Get current code
	currentCode := service.GetCurrentCode(secret)

	// Current code should verify with window
	if !service.VerifyWithWindow(secret, currentCode, 1) {
		t.Error("VerifyWithWindow should return true for current code")
	}

	// Test with wrong code
	wrongCode := "999999"
	if currentCode != wrongCode && service.VerifyWithWindow(secret, wrongCode, 1) {
		t.Error("VerifyWithWindow should return false for invalid code")
	}
}

func TestGetCurrentCode(t *testing.T) {
	service := NewService("TestIssuer")
	secret := "JBSWY3DPEHPK3PXP"

	code := service.GetCurrentCode(secret)

	// Code should be 6 digits
	if len(code) != 6 {
		t.Errorf("expected code length 6, got %d", len(code))
	}

	// Code should only contain digits
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("code should only contain digits, got: %s", code)
		}
	}
}

func TestGetCurrentUnixTimestamp(t *testing.T) {
	// Test that timestamp changes over time
	ts1 := getCurrentUnixTimestamp()
	time.Sleep(100 * time.Millisecond)
	ts2 := getCurrentUnixTimestamp()

	// Timestamps should be approximately the same (difference should be small)
	diff := ts2 - ts1
	if diff < 0 || diff > 1 {
		t.Errorf("timestamp difference should be 0 or 1 (within 100ms), got %d", diff)
	}
}

func TestTOTPInterval(t *testing.T) {
	if totpInterval != 30 {
		t.Errorf("TOTP interval should be 30 seconds, got %d", totpInterval)
	}
}
