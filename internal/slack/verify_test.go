package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"
)

const testSecret = "8f742231b10e8888abcd99yyyzzz85a5"

// sign produces a valid Slack signature for the given timestamp and body.
func sign(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "v0:%s:%s", timestamp, body)
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerify_ValidSignature(t *testing.T) {
	fixedNow := time.Unix(1_700_000_000, 0)
	v := NewVerifier(testSecret)
	v.now = func() time.Time { return fixedNow }

	body := []byte("command=%2Fdraftinfo&text=")
	ts := strconv.FormatInt(fixedNow.Unix(), 10)
	sig := sign(testSecret, ts, body)

	if err := v.Verify(ts, sig, body); err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}
}

func TestVerify_BadSignature(t *testing.T) {
	fixedNow := time.Unix(1_700_000_000, 0)
	v := NewVerifier(testSecret)
	v.now = func() time.Time { return fixedNow }

	body := []byte("command=%2Fdraftinfo")
	ts := strconv.FormatInt(fixedNow.Unix(), 10)
	// Sign with the wrong secret.
	sig := sign("wrong-secret", ts, body)

	if err := v.Verify(ts, sig, body); err == nil {
		t.Fatal("expected signature mismatch error, got nil")
	}
}

func TestVerify_ExpiredTimestamp(t *testing.T) {
	fixedNow := time.Unix(1_700_000_000, 0)
	v := NewVerifier(testSecret)
	v.now = func() time.Time { return fixedNow }

	// Timestamp 10 minutes in the past — beyond the 5 minute window.
	oldTs := strconv.FormatInt(fixedNow.Add(-10*time.Minute).Unix(), 10)
	body := []byte("command=%2Fdraftinfo")
	sig := sign(testSecret, oldTs, body)

	if err := v.Verify(oldTs, sig, body); err == nil {
		t.Fatal("expected stale timestamp error, got nil")
	}
}

func TestVerify_InvalidTimestampHeader(t *testing.T) {
	v := NewVerifier(testSecret)
	if err := v.Verify("not-a-number", "v0=abc", []byte("x")); err == nil {
		t.Fatal("expected error for non-numeric timestamp, got nil")
	}
}
