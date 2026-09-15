package durable_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type SanitizationPolicySuite struct {
	suite.Suite
}

func TestSanitizationPolicySuite(t *testing.T) {
	suite.Run(t, new(SanitizationPolicySuite))
}

func (s *SanitizationPolicySuite) TestCatalogPatterns() {
	scenarios := []struct {
		name         string
		content      string
		wantPattern  string
		wantRedacted bool
	}{
		{
			name:         "should redact PEM private key",
			content:      "before\n-----BEGIN RSA PRIVATE KEY-----\nMIIBVgIBADANBgkqhkiG9w0BAQ\n-----END RSA PRIVATE KEY-----\nafter",
			wantPattern:  "pem_private_key",
			wantRedacted: true,
		},
		{
			name:         "should not redact plain text without PEM markers",
			content:      "no secret here, just prose about keys",
			wantRedacted: false,
		},
		{
			name:         "should redact GitHub token prefix",
			content:      "token is ghp_1234567890abcdefGHIJ in config",
			wantPattern:  "provider_token",
			wantRedacted: true,
		},
		{
			name:         "should not redact short string resembling prefix",
			content:      "the sk- word here is not a token",
			wantRedacted: false,
		},
		{
			name:         "should redact AWS access key id",
			content:      "AKIA" + "IOSFODNN7EXAMPLE" + " stored in env",
			wantPattern:  "provider_token",
			wantRedacted: true,
		},
		{
			name:         "should redact JWT",
			content:      "auth is eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U in logs",
			wantPattern:  "jwt",
			wantRedacted: true,
		},
		{
			name:         "should not redact plain sentence with dots",
			content:      "this.is.not.a.jwt.token.at.all",
			wantRedacted: false,
		},
		{
			name:         "should redact Authorization header",
			content:      "request had Authorization: Bearer abc123tokenvalue",
			wantPattern:  "authorization_header",
			wantRedacted: true,
		},
		{
			name:         "should not redact unrelated header",
			content:      "request had Content-Type: application/json",
			wantRedacted: false,
		},
		{
			name:         "should redact dotenv secret value",
			content:      "config dump:\nDATABASE_PASSWORD=supersecretvalue\nend of dump",
			wantPattern:  "dotenv_secret",
			wantRedacted: true,
		},
		{
			name:         "should not redact unrelated env key",
			content:      "APP_NAME=orchestrator",
			wantRedacted: false,
		},
		{
			name:         "should redact connection string with credential",
			content:      "connect to postgres://admin:hunter2@db.internal:5432/app",
			wantPattern:  "connection_string_credential",
			wantRedacted: true,
		},
		{
			name:         "should not redact connection string without credential",
			content:      "connect to postgres://db.internal:5432/app",
			wantRedacted: false,
		},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			policy := durable.SanitizationPolicy{}

			result, err := policy.Sanitize(sc.content, durable.SanitizationConfig{})

			s.Require().NoError(err)
			if sc.wantRedacted {
				s.Require().NotEmpty(result.Redactions)
				s.Equal(sc.wantPattern, result.Redactions[0].Pattern)
				s.NotEqual(sc.content, result.Content)
			} else {
				s.Empty(result.Redactions)
				s.Equal(sc.content, result.Content)
			}
		})
	}
}

func (s *SanitizationPolicySuite) TestSanitizeRefusesWhenSecretIsNotIsolable() {
	policy := durable.SanitizationPolicy{}

	_, err := policy.Sanitize("-----BEGIN RSA PRIVATE KEY-----\nMIIBVgIBADANBgkqhkiG9w0BAQ\n-----END RSA PRIVATE KEY-----", durable.SanitizationConfig{})

	s.True(errors.Is(err, durable.ErrSecretNotRedactable))
}

func (s *SanitizationPolicySuite) TestSanitizeIsExtensibleByConfig() {
	custom, err := durable.NewSanitizationPattern("internal_ticket_id", `\bTICKET-\d{5}\b`)
	s.Require().NoError(err)

	policy := durable.SanitizationPolicy{}
	result, err := policy.Sanitize("reference TICKET-12345 in the report", durable.SanitizationConfig{
		ExtraPatterns: []durable.SanitizationPattern{custom},
	})

	s.Require().NoError(err)
	s.Require().Len(result.Redactions, 1)
	s.Equal("internal_ticket_id", result.Redactions[0].Pattern)
}

func (s *SanitizationPolicySuite) TestSanitizeIsDeterministicAcrossRuns() {
	policy := durable.SanitizationPolicy{}
	content := "token ghp_1234567890abcdefGHIJ and Authorization: Bearer abc123tokenvalue"

	first, err := policy.Sanitize(content, durable.SanitizationConfig{})
	s.Require().NoError(err)
	second, err := policy.Sanitize(content, durable.SanitizationConfig{})
	s.Require().NoError(err)

	s.Equal(first, second)
}
