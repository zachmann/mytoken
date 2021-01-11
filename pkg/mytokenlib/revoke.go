package mytokenlib

import (
	"github.com/zachmann/mytoken/internal/httpClient"
	"github.com/zachmann/mytoken/internal/server/endpoints/revocation/pkg"
	"github.com/zachmann/mytoken/pkg/model"
)

func (my *Mytoken) Revoke(superToken, oidcIssuer string, recursive bool) error {
	req := pkg.RevocationRequest{
		Token:      superToken,
		Recursive:  recursive,
		OIDCIssuer: oidcIssuer,
	}
	resp, err := httpClient.Do().R().SetBody(req).SetError(&model.APIError{}).Post(my.RevocationEndpoint)
	if err != nil {
		return newMytokenErrorFromError("error while sending http request", err)
	}
	if e := resp.Error(); e != nil {
		if errRes := e.(*model.APIError); errRes != nil && len(errRes.Error) > 0 {
			return &MytokenError{
				err:          errRes.Error,
				errorDetails: errRes.ErrorDescription,
			}
		}
	}
	return nil
}
