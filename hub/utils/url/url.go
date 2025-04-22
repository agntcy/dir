package url

import (
	"fmt"
	"net/url"
)

func ValidateSecureURL(u string) error {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be HTTPS")
	}

	return nil
}
