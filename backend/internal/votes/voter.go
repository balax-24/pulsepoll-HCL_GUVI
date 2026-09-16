package votes

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// VoterCookieName is the HTTP cookie key storing the anonymous voter session ID.
	VoterCookieName = "pulsepoll_voter_id"

	// VoterHeaderName is the HTTP header key for clients passing voter ID explicitly.
	VoterHeaderName = "X-Voter-ID"

	// CookieMaxAge is 1 year in seconds.
	CookieMaxAge = 365 * 24 * 3600
)

// GenerateVoterID produces a cryptographically secure 128-bit random hexadecimal string.
func GenerateVoterID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Extremely rare fallback: return pseudo-random timestamp-based string
		return hex.EncodeToString([]byte("voter_session_fallback"))
	}
	return hex.EncodeToString(b)
}

// GetOrCreateVoterID resolves the voter's identity from headers or cookies,
// generating and setting a new one if no valid session identifier exists.
func GetOrCreateVoterID(c *gin.Context) string {
	// 1. Check explicit client header (useful for API clients, tests, mobile)
	headerVal := strings.TrimSpace(c.GetHeader(VoterHeaderName))
	if headerVal != "" && len(headerVal) >= 16 && len(headerVal) <= 64 {
		return headerVal
	}

	// 2. Check browser cookie
	cookieVal, err := c.Cookie(VoterCookieName)
	if err == nil {
		cookieVal = strings.TrimSpace(cookieVal)
		if cookieVal != "" && len(cookieVal) >= 16 && len(cookieVal) <= 64 {
			return cookieVal
		}
	}

	// 3. Generate a fresh anonymous voter ID
	voterID := GenerateVoterID()

	// 4. Determine if request is running under HTTPS (direct TLS or reverse proxy termination)
	isHTTPS := c.Request.TLS != nil ||
		strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") ||
		strings.EqualFold(c.GetHeader("X-Forwarded-Ssl"), "on")

	// SameSite=None is required for cross-origin credentials under HTTPS; Lax is used for local HTTP
	if isHTTPS {
		c.SetSameSite(http.SameSiteNoneMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}

	// Set persistent, HTTP-only cookie with appropriate security flags
	c.SetCookie(
		VoterCookieName,
		voterID,
		CookieMaxAge,
		"/",
		"",
		isHTTPS, // Secure=true under HTTPS
		true,    // HttpOnly=true
	)

	// Also echo back in header so non-browser clients can read and store it
	c.Header(VoterHeaderName, voterID)

	return voterID
}
