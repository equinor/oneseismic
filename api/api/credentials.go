/*
* Serialization support for credentials
*/
package api

import (
	"fmt"
	"log"
	"strings"

	"github.com/equinor/oneseismic/api/internal/auth"
)

func EncodeCredentials(tokens *auth.Tokens, keys map[string]string) (string, error) {
	pid   := keys["pid"]
	token := keys["Authorization"]
	if token != "" {
		tok, err := (*tokens).GetOnbehalf(token); if err != nil {
			log.Printf("pid=%s %v", pid, err)
			return "", NewIllegalInputError("Failed to construct OBO")
		} else {
			return fmt.Sprintf("OBO|%s", tok), nil
		}
	}
	query := keys["url-query"]
	if query != "" {
		return fmt.Sprintf("SAAS|%s", query), nil
	}
	return "", nil
}

/*
* The complex (over-engineered?) return-value is for future support
* for authentication-mechanisms more complex than a single token, eg.
* username/password
*/
func DecodeCredentials(credString string) (string, map[string]string, error) {
	parts := strings.Split(credString, "|")
	if len(parts) != 2 {
		log.Printf("Failed to parse credentials-string %v", credString)
		return "",map[string]string{}, NewIllegalInputError(
			fmt.Sprintf("Failed to parse credentials-string %v", credString))
	}
	return strings.ToLower(parts[0]), map[string]string{"token": parts[1]}, nil
}
