package restrictions

import (
	"github.com/jmoiron/sqlx"

	"github.com/oidc-mytoken/server/internal/db"
	"github.com/oidc-mytoken/server/shared/supertoken/pkg/stid"
)

// UsedRestriction is a type for a restriction that has been used and additionally has information how often is has been used
type UsedRestriction struct {
	Restriction
	UsagesATDone    *int64 `json:"usages_AT_done,omitempty"`
	UsagesOtherDone *int64 `json:"usages_other_done,omitempty"`
}

func (r Restrictions) ToUsedRestrictions(id stid.STID) (ur []UsedRestriction, err error) {
	var u UsedRestriction
	for _, rr := range r {
		u, err = rr.ToUsedRestriction(id)
		if err != nil {
			return
		}
		ur = append(ur, u)
	}
	return
}

func (r Restriction) ToUsedRestriction(id stid.STID) (UsedRestriction, error) {
	ur := UsedRestriction{
		Restriction: r,
	}
	err := db.Transact(func(tx *sqlx.Tx) error {
		at, err := r.getATUsageCounts(tx, id)
		if err != nil {
			return err
		}
		ur.UsagesATDone = at
		other, err := r.getOtherUsageCounts(tx, id)
		ur.UsagesOtherDone = other
		return err
	})
	return ur, err
}