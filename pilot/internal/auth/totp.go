package auth

import (
	"github.com/pquerna/otp/totp"
)

// GenerateTOTP creates a new TOTP secret for the given email address.
// It returns the base32-encoded secret, the otpauth:// URL (for QR codes), and any error.
func GenerateTOTP(email string) (secret string, url string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Postpilot",
		AccountName: email,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// ValidateTOTP checks a 6-digit TOTP code against the given base32-encoded secret.
// Returns true if the code is valid for the current time window.
func ValidateTOTP(code, secret string) bool {
	return totp.Validate(code, secret)
}
