package supertoken

import (
	"github.com/oidc-mytoken/server/shared/supertoken/restrictions"
)

// UsedSuperToken is a type for a SuperToken that has been used, it additionally has information how often it has been used
type UsedSuperToken struct {
	SuperToken
	Restrictions []restrictions.UsedRestriction `json:"restrictions,omitempty"`
}

func (st *SuperToken) ToUsedSuperToken() (*UsedSuperToken, error) {
	ust := &UsedSuperToken{
		SuperToken: *st,
	}
	var err error
	ust.Restrictions, err = st.Restrictions.ToUsedRestrictions(st.ID)
	return ust, err
}