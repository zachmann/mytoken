package pkg

import (
	"github.com/oidc-mytoken/server/pkg/api/v0"
	"github.com/oidc-mytoken/server/shared/model"
	"github.com/oidc-mytoken/server/shared/mytoken/restrictions"
)

// MytokenResponse is a response to a mytoken request
type MytokenResponse struct {
	api.MytokenResponse `json:",inline"`
	MytokenType         model.ResponseType        `json:"mytoken_type"`
	Restrictions        restrictions.Restrictions `json:"restrictions,omitempty"`
}
