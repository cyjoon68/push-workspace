package infra

import (
	"testing"

	"github.com/google/uuid"
)

func TestKeyCipherRoundTrip(t *testing.T) {
	c, err := NewKeyCipher("test-master-key")
	if err != nil {
		t.Fatal(err)
	}
	uid := uuid.New()
	ct, nonce, err := c.Encrypt("secret-token", uid, "google")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := c.Decrypt(ct, nonce, uid, "google")
	if err != nil {
		t.Fatal(err)
	}
	if pt != "secret-token" {
		t.Fatalf("got %q", pt)
	}
}

func TestKeyCipherDecryptWrongPurpose(t *testing.T) {
	c, _ := NewKeyCipher("test-master-key")
	uid := uuid.New()
	ct, nonce, _ := c.Encrypt("secret-token", uid, "google")
	if _, err := c.Decrypt(ct, nonce, uid, "openai"); err == nil {
		t.Fatal("expected AAD mismatch error")
	}
	if _, err := c.Decrypt(ct, nonce, uuid.New(), "google"); err == nil {
		t.Fatal("expected user mismatch error")
	}
}
