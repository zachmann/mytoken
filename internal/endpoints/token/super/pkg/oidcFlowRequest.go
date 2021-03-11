package pkg

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/oidc-mytoken/server/pkg/model"
	"github.com/oidc-mytoken/server/shared/supertoken/capabilities"
	"github.com/oidc-mytoken/server/shared/supertoken/restrictions"
)

// OIDCFlowRequest holds the request for an OIDC Flow request
type OIDCFlowRequest struct {
	Issuer               string                    `json:"oidc_issuer"`
	GrantType            model.GrantType           `json:"grant_type"`
	OIDCFlow             model.OIDCFlow            `json:"oidc_flow"`
	Restrictions         restrictions.Restrictions `json:"restrictions"`
	Capabilities         capabilities.Capabilities `json:"capabilities"`
	SubtokenCapabilities capabilities.Capabilities `json:"subtoken_capabilities"`
	Name                 string                    `json:"name"`
	ResponseType         model.ResponseType        `json:"response_type"`
	redirectType         string
}

// NewOIDCFlowRequest creates a new OIDCFlowRequest with default values where they can be omitted
func NewOIDCFlowRequest() *OIDCFlowRequest {
	return &OIDCFlowRequest{
		Capabilities: capabilities.Capabilities{capabilities.CapabilityAT},
		ResponseType: model.ResponseTypeToken,
		redirectType: redirectTypeWeb,
	}
}

// MarshalJSON implements the json.Marshaler interface
func (r OIDCFlowRequest) MarshalJSON() ([]byte, error) {
	type ofr OIDCFlowRequest
	o := struct {
		ofr
		RedirectType string `json:"redirect_type,omitempty"`
	}{
		ofr:          ofr(r),
		RedirectType: r.redirectType,
	}
	return json.Marshal(o)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (r *OIDCFlowRequest) UnmarshalJSON(data []byte) error {
	type ofr OIDCFlowRequest
	o := struct {
		ofr
		RedirectType string `json:"redirect_type"`
	}{
		ofr: ofr(*NewOIDCFlowRequest()),
	}
	o.RedirectType = o.redirectType
	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}
	o.redirectType = o.RedirectType
	*r = OIDCFlowRequest(o.ofr)
	if r.SubtokenCapabilities != nil && !r.Capabilities.Has(capabilities.CapabilityCreateST) {
		r.SubtokenCapabilities = nil
	}
	return nil
}

func (r OIDCFlowRequest) ToAuthCodeFlowRequest() AuthCodeFlowRequest {
	return AuthCodeFlowRequest{
		OIDCFlowRequest: r,
		RedirectType:    r.redirectType,
	}
}

// Scan implements the sql.Scanner interface
func (r *OIDCFlowRequest) Scan(src interface{}) error {
	v, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("bad []byte type assertion")
	}
	return json.Unmarshal(v, r)
}

// Value implements the driver.Valuer interface
func (r OIDCFlowRequest) Value() (driver.Value, error) {
	return json.Marshal(r)
}
