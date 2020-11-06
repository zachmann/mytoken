package dbModels

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	uuid "github.com/satori/go.uuid"
	"github.com/zachmann/mytoken/internal/db"
)

type AccessToken struct {
	Token   string
	IP      string
	Comment string
	STID    uuid.UUID

	Scopes    []string
	Audiences []string
}

type accessToken struct {
	Token   string
	IP      string `db:"ip_created"`
	Comment sql.NullString
	STID    uuid.UUID `db:"ST_id"`
}

func (t *AccessToken) toDBObject() accessToken {
	return accessToken{
		Token:   t.Token,
		IP:      t.IP,
		Comment: db.NewNullString(t.Comment),
		STID:    t.STID,
	}
}

func (t *AccessToken) getDBAttributes(atID uint64) (attrs []AccessTokenAttribute, err error) {
	var scopeAttrID uint64
	var audAttrID uint64
	if err = db.DB().QueryRow(`SELECT id FROM Attributes WHERE attribute=?`, "scope").Scan(scopeAttrID); err != nil {
		return
	}
	if err = db.DB().QueryRow(`SELECT id FROM Attributes WHERE attribute=?`, "audience").Scan(audAttrID); err != nil {
		return
	}
	for _, s := range t.Scopes {
		attrs = append(attrs, AccessTokenAttribute{
			ATID:   atID,
			AttrID: scopeAttrID,
			Attr:   s,
		})
	}
	for _, a := range t.Audiences {
		attrs = append(attrs, AccessTokenAttribute{
			ATID:   atID,
			AttrID: audAttrID,
			Attr:   a,
		})
	}
	return
}

func (t *AccessToken) Store() error {
	return db.Transact(func(tx *sqlx.Tx) error {
		store := t.toDBObject()
		res, err := tx.NamedExec(`INSERT INTO AccessTokens (token, ip_created, comment, ST_id) VALUES (:token, :ip_created, :comment, :ST_id)`, store)
		if err != nil {
			return err
		}
		if len(t.Scopes) > 0 || len(t.Audiences) > 0 {
			atID, err := res.LastInsertId()
			if err != nil {
				return err
			}
			attrs, err := t.getDBAttributes(uint64(atID))
			if _, err := tx.NamedExec(`INSERT INTO AT_Attributes (AT_id, attribute_id, attribute) VALUES (:AT_id, :attribute_id, :attribute)`, attrs); err != nil {
				return err
			}
		}
		return nil
	})
}