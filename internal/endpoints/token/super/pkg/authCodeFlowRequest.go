package pkg

import (
	"encoding/json"
)

// Redirect types
const (
	redirectTypeWeb    = "web"
	redirectTypeNative = "native"
)

// AuthCodeFlowRequest holds a authorization code flow request
type AuthCodeFlowRequest struct {
	OIDCFlowRequest
	RedirectType string `json:"redirect_type"`
}

// Native checks if the request is native
func (r *AuthCodeFlowRequest) Native() bool {
	if r.RedirectType == redirectTypeNative {
		return true
	}
	return false
}

// UnmarshalJSON implements the json unmarshaler interface
func (r *AuthCodeFlowRequest) UnmarshalJSON(data []byte) error {
	var tmp OIDCFlowRequest
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*r = tmp.ToAuthCodeFlowRequest()
	return nil
}
