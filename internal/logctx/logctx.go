// Package logctx carries request-scoped audit metadata from the component that
// detects a failure to the access-log middleware that reports it. Every
// function here only stores or reads gin context values; nothing in this
// package can change a status code, a response body, or control flow.
package logctx

import "github.com/gin-gonic/gin"

// KeyReason holds the machine-readable cause of a rejected or failed request,
// KeyDetail the underlying error text when one is in hand.
const (
	KeyReason = "reason"
	KeyDetail = "detail"
)

// KeyRule holds the whitelist/blacklist/greylist rule that matched, as set by
// CheckIPMiddleware. It is kept here so every audit-log key is defined once.
const KeyRule = "rule"

// Reason is the machine-readable cause of a rejected or failed request.
type Reason string

// Reasons recorded along the request chain. AccessLog falls back to
// ReasonUnattributed for any 4xx/5xx that records none.
const (
	// Authentication, internal/middleware/auth.go.
	ReasonMissingBearerToken Reason = "missing_bearer_token"
	ReasonInvalidToken       Reason = "invalid_token"
	ReasonInvalidSession     Reason = "invalid_session"
	ReasonInsufficientRole   Reason = "insufficient_role"

	// IP access control, internal/middleware/checkip.go.
	ReasonIPRuleLookupFailed Reason = "ip_rule_lookup_failed"
	ReasonIPBlacklisted      Reason = "ip_blacklisted"
	ReasonIPNotWhitelisted   Reason = "ip_not_whitelisted"
	ReasonUnknownIPRule      Reason = "unknown_ip_rule"

	// Rate limiting, internal/middleware/ratelimit.go.
	ReasonRateLimitUnavailable Reason = "ratelimit_unavailable"
	ReasonRateLimitExceeded    Reason = "ratelimit_exceeded"

	// User handlers, internal/handler/user_handler.go.
	ReasonSMTPNotConfigured   Reason = "smtp_not_configured"
	ReasonInvalidBody         Reason = "invalid_body"
	ReasonCodeRecentlySent    Reason = "code_recently_sent"
	ReasonInvalidEmail        Reason = "invalid_email"
	ReasonSendCodeFailed      Reason = "send_code_failed"
	ReasonCodeRequired        Reason = "code_required"
	ReasonVerifyCodeFailed    Reason = "verify_code_failed"
	ReasonInvalidCode         Reason = "invalid_code"
	ReasonInvalidInviteToken  Reason = "invalid_invite_token"
	ReasonMissingCredentials  Reason = "missing_credentials"
	ReasonInvalidEmailCode    Reason = "invalid_email_code"
	ReasonCreateUserFailed    Reason = "create_user_failed"
	ReasonUserLookupFailed    Reason = "user_lookup_failed"
	ReasonInvalidCredentials  Reason = "invalid_credentials"
	ReasonSignTokenFailed     Reason = "sign_token_failed"
	ReasonCreateSessionFailed Reason = "create_session_failed"
	ReasonDeleteSessionFailed Reason = "delete_session_failed"

	// Admin and token handlers, internal/handler/admin_handler.go and
	// internal/handler/token_handler.go.
	ReasonInvalidIP           Reason = "invalid_ip"
	ReasonIPRuleExists        Reason = "ip_rule_exists"
	ReasonIPRuleNotFound      Reason = "ip_rule_not_found"
	ReasonAddIPRuleFailed     Reason = "add_ip_rule_failed"
	ReasonListIPRulesFailed   Reason = "list_ip_rules_failed"
	ReasonDeleteIPRuleFailed  Reason = "delete_ip_rule_failed"
	ReasonListUsersFailed     Reason = "list_users_failed"
	ReasonRestockTokensFailed Reason = "restock_tokens_failed"
	ReasonFlushTokensFailed   Reason = "flush_tokens_failed"

	// Recorded by the access log itself, never by a producer.
	ReasonUnattributed  Reason = "unattributed"
	ReasonRouteNotFound Reason = "route_not_found"
)

// SetReason records why the current request is being rejected or failed. The
// innermost component to call it wins, which is the one that writes the status.
func SetReason(c *gin.Context, reason Reason) {
	if reason != "" {
		c.Set(KeyReason, string(reason))
	}
}

// SetReasonDetail records a reason together with the error that caused it. A
// nil error records the reason alone.
func SetReasonDetail(c *gin.Context, reason Reason, err error) {
	SetReason(c, reason)
	if err != nil {
		c.Set(KeyDetail, err.Error())
	}
}

// ReasonOf returns the recorded reason, or "" when none was set.
func ReasonOf(c *gin.Context) string { return c.GetString(KeyReason) }

// DetailOf returns the recorded error text, or "" when none was set.
func DetailOf(c *gin.Context) string { return c.GetString(KeyDetail) }
