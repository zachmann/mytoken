package polling

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"

	"github.com/oidc-mytoken/server/internal/db/dbrepo/mytokenrepo/transfercoderepo"
	response "github.com/oidc-mytoken/server/internal/endpoints/token/mytoken/pkg"
	"github.com/oidc-mytoken/server/internal/model"
	"github.com/oidc-mytoken/server/internal/utils/ctxUtils"
	"github.com/oidc-mytoken/server/pkg/api/v0"
	mytoken "github.com/oidc-mytoken/server/shared/mytoken/pkg"
)

// HandlePollingCode handles a request on the polling endpoint
func HandlePollingCode(ctx *fiber.Ctx) error {
	req := response.PollingCodeRequest{}
	if err := json.Unmarshal(ctx.Body(), &req); err != nil {
		return model.ErrorToBadRequestErrorResponse(err).Send(ctx)
	}
	return handlePollingCode(req, *ctxUtils.ClientMetaData(ctx)).Send(ctx)
}

func handlePollingCode(req response.PollingCodeRequest, networkData api.ClientMetaData) *model.Response {
	pollingCode := req.PollingCode
	log.WithField("polling_code", pollingCode).Debug("Handle polling code")
	pollingCodeStatus, err := transfercoderepo.CheckTransferCode(nil, pollingCode)
	if err != nil {
		return model.ErrorToInternalServerErrorResponse(err)
	}
	if !pollingCodeStatus.Found {
		log.WithField("polling_code", pollingCode).Debug("Polling code not known")
		return &model.Response{
			Status:   fiber.StatusUnauthorized,
			Response: api.APIErrorBadTransferCode,
		}
	}
	if pollingCodeStatus.ConsentDeclined {
		log.WithField("polling_code", pollingCode).Debug("Consent declined")
		return &model.Response{
			Status:   fiber.StatusUnauthorized,
			Response: api.APIErrorConsentDeclined,
		}
	}
	if pollingCodeStatus.Expired {
		log.WithField("polling_code", pollingCode).Debug("Polling code expired")
		return &model.Response{
			Status:   fiber.StatusUnauthorized,
			Response: api.APIErrorTransferCodeExpired,
		}
	}
	token, err := transfercoderepo.PopTokenForTransferCode(nil, pollingCode, networkData)
	if err != nil {
		log.WithError(err).Error()
		return model.ErrorToInternalServerErrorResponse(err)
	}
	if token == "" {
		return &model.Response{
			Status:   fiber.StatusPreconditionRequired,
			Response: api.APIErrorAuthorizationPending,
		}
	}
	mt, err := mytoken.ParseJWT(token)
	if err != nil {
		return model.ErrorToInternalServerErrorResponse(err)
	}
	log.Tracef("The JWT was parsed as '%+v'", mt)
	res, err := mt.ToTokenResponse(pollingCodeStatus.ResponseType, networkData, token)
	if err != nil {
		return model.ErrorToInternalServerErrorResponse(err)
	}
	return &model.Response{
		Status:   fiber.StatusOK,
		Response: res,
	}
}
