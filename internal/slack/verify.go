// Package slack handles Slack slash-command parsing, request verification, and
// response formatting.
package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// maxTimestampSkew is the largest difference we tolerate between the request
// timestamp and now. Slack recommends 5 minutes to guard against replay.
const maxTimestampSkew = 5 * time.Minute

// Verifier validates that a request genuinely came from Slack using the
// signing-secret HMAC scheme:
//
//	basestring = "v0:" + timestamp + ":" + rawBody
//	expected   = "v0=" + hex(HMAC_SHA256(signingSecret, basestring))
//
// See https://api.slack.com/authentication/verifying-requests-from-slack
type Verifier struct {
	signingSecret []byte
	now           func() time.Time // injectable for tests
}

// NewVerifier returns a Verifier for the given signing secret.
func NewVerifier(signingSecret string) *Verifier {
	return &Verifier{
		signingSecret: []byte(signingSecret),
		now:           time.Now,
	}
}

// Verify checks the X-Slack-Signature / X-Slack-Request-Timestamp headers
// against the raw request body. It returns an error if the timestamp is stale
// or the signature does not match.
func (verifier *Verifier) Verify(timestamp, signature string, body []byte) error {
	unixSeconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp header: %w", err)
	}

	skew := verifier.now().Sub(time.Unix(unixSeconds, 0))
	if skew < 0 {
		skew = -skew
	}
	if skew > maxTimestampSkew {
		return fmt.Errorf("timestamp outside allowed window (%s)", skew)
	}

	mac := hmac.New(sha256.New, verifier.signingSecret)
	fmt.Fprintf(mac, "v0:%s:%s", timestamp, body)
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison to avoid leaking the signature via timing.
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}
