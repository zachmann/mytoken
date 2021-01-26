package pkg

import (
	"github.com/oidc-mytoken/server/internal/supertoken/capabilities"
	"github.com/oidc-mytoken/server/internal/supertoken/restrictions"
	"github.com/oidc-mytoken/server/pkg/model"
)

// SuperTokenResponse is a response to a super token request
type SuperTokenResponse struct {
	SuperToken           string                    `json:"super_token,omitempty"`
	SuperTokenType       model.ResponseType        `json:"super_token_type"`
	TransferCode         string                    `json:"transfer_code,omitempty"`
	ExpiresIn            uint64                    `json:"expires_in,omitempty"`
	Restrictions         restrictions.Restrictions `json:"restrictions,omitempty"`
	Capabilities         capabilities.Capabilities `json:"capabilities,omitempty"`
	SubtokenCapabilities capabilities.Capabilities `json:"subtoken_capabilities,omitempty"`
}