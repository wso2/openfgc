package me

import "testing"

func TestIsConsentOwnedByUser(t *testing.T) {
	body := []byte(`{"authorizations":[{"userId":"user-1"}]}`)
	if !IsConsentOwnedByUser(body, "user-1") {
		t.Fatal("expected matching authorization to be owned")
	}
	if IsConsentOwnedByUser(body, "user-2") {
		t.Fatal("expected foreign authorization to be rejected")
	}
	if IsConsentOwnedByUser([]byte(`{"authorizations":[]}`), "user-1") {
		t.Fatal("expected missing authorization to be rejected")
	}
}
