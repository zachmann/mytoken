package access

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	"github.com/zachmann/mytoken/internal/config"
	"github.com/zachmann/mytoken/internal/db"
	"github.com/zachmann/mytoken/internal/db/dbrepo/accesstokenrepo"
	dbhelper "github.com/zachmann/mytoken/internal/db/dbrepo/supertokenrepo/supertokenrepohelper"
	request "github.com/zachmann/mytoken/internal/endpoints/token/access/pkg"
	"github.com/zachmann/mytoken/internal/model"
	"github.com/zachmann/mytoken/internal/oidc/refresh"
	"github.com/zachmann/mytoken/internal/supertoken/capabilities"
	eventService "github.com/zachmann/mytoken/internal/supertoken/event"
	event "github.com/zachmann/mytoken/internal/supertoken/event/pkg"
	supertoken "github.com/zachmann/mytoken/internal/supertoken/pkg"
	"github.com/zachmann/mytoken/internal/supertoken/restrictions"
	"github.com/zachmann/mytoken/internal/utils"
	"github.com/zachmann/mytoken/internal/utils/ctxUtils"
	"github.com/zachmann/mytoken/internal/utils/oidcUtils"
)

// HandleAccessTokenEndpoint handles request on the access token endpoint
func HandleAccessTokenEndpoint(ctx *fiber.Ctx) error {
	log.Debug("Handle access token request")
	req := request.AccessTokenRequest{}
	if err := json.Unmarshal(ctx.Body(), &req); err != nil {
		return model.ErrorToBadRequestErrorResponse(err).Send(ctx)
	}
	log.Trace("Parsed access token request")

	if req.GrantType != model.GrantTypeSuperToken {
		res := model.Response{
			Status:   fiber.StatusBadRequest,
			Response: model.APIErrorUnsupportedGrantType,
		}
		return res.Send(ctx)
	}
	log.Trace("Checked grant type")

	st, err := supertoken.ParseJWT(string(req.SuperToken))
	if err != nil {
		return (&model.Response{
			Status:   fiber.StatusUnauthorized,
			Response: model.InvalidTokenError(err.Error()),
		}).Send(ctx)
	}
	log.Trace("Parsed super token")

	revoked, dbErr := dbhelper.CheckTokenRevoked(st.ID)
	if dbErr != nil {
		return model.ErrorToInternalServerErrorResponse(dbErr).Send(ctx)
	}
	if revoked {
		return (&model.Response{
			Status:   fiber.StatusUnauthorized,
			Response: model.InvalidTokenError("not a valid token"),
		}).Send(ctx)
	}
	log.Trace("Checked token not revoked")

	if ok := st.Restrictions.VerifyForAT(nil, ctx.IP(), st.ID); !ok {
		return (&model.Response{
			Status:   fiber.StatusForbidden,
			Response: model.APIErrorUsageRestricted,
		}).Send(ctx)
	}
	log.Trace("Checked super token restrictions")
	if ok := st.VerifyCapabilities(capabilities.CapabilityAT); !ok {
		res := model.Response{
			Status:   fiber.StatusForbidden,
			Response: model.APIErrorInsufficientCapabilities,
		}
		return res.Send(ctx)
	}
	log.Trace("Checked super token capabilities")
	if req.Issuer != st.OIDCIssuer {
		res := model.Response{
			Status:   fiber.StatusBadRequest,
			Response: model.BadRequestError("token not for specified issuer"),
		}
		return res.Send(ctx)
	}
	log.Trace("Checked issuer")

	return handleAccessTokenRefresh(st, req, *ctxUtils.ClientMetaData(ctx)).Send(ctx)
}

func handleAccessTokenRefresh(st *supertoken.SuperToken, req request.AccessTokenRequest, networkData model.ClientMetaData) *model.Response {
	provider, ok := config.Get().ProviderByIssuer[req.Issuer]
	if !ok {
		return &model.Response{
			Status:   fiber.StatusBadRequest,
			Response: model.APIErrorUnknownIssuer,
		}
	}

	scopes := strings.Join(provider.Scopes, " ") // default if no restrictions apply
	auds := ""                                   // default if no restrictions apply
	var usedRestriction *restrictions.Restriction
	if len(st.Restrictions) > 0 {
		possibleRestrictions := st.Restrictions.GetValidForAT(nil, networkData.IP, st.ID).WithScopes(utils.SplitIgnoreEmpty(req.Scope, " ")).WithAudiences(utils.SplitIgnoreEmpty(req.Audience, " "))
		if len(possibleRestrictions) == 0 {
			return &model.Response{
				Status:   fiber.StatusBadRequest,
				Response: model.APIErrorUsageRestricted,
			}
		}
		usedRestriction = &possibleRestrictions[0]
		if len(req.Scope) > 0 {
			scopes = req.Scope
		} else if len(usedRestriction.Scope) > 0 {
			scopes = usedRestriction.Scope
		}
		if len(req.Audience) != 0 {
			auds = req.Audience
		} else if len(usedRestriction.Audiences) > 0 {
			auds = strings.Join(usedRestriction.Audiences, " ")
		}
	}
	rt, rtFound, dbErr := dbhelper.GetRefreshToken(st.ID, req.SuperToken)
	if dbErr != nil {
		return model.ErrorToInternalServerErrorResponse(dbErr)
	}
	if !rtFound {
		return &model.Response{
			Status:   fiber.StatusUnauthorized,
			Response: model.InvalidTokenError("No refresh token attached"),
		}
	}

	oidcRes, oidcErrRes, err := refresh.RefreshFlowAndUpdateDB(provider, string(req.SuperToken), rt, scopes, auds)
	if err != nil {
		return model.ErrorToInternalServerErrorResponse(err)
	}
	if oidcErrRes != nil {
		return &model.Response{
			Status:   oidcErrRes.Status,
			Response: model.OIDCError(oidcErrRes.Error, oidcErrRes.ErrorDescription),
		}
	}
	retScopes := scopes
	if len(oidcRes.Scopes) > 0 {
		retScopes = oidcRes.Scopes
	}
	retAudiences, _ := oidcUtils.GetAudiencesFromJWT(oidcRes.AccessToken)
	at := accesstokenrepo.AccessToken{
		Token:     oidcRes.AccessToken,
		IP:        networkData.IP,
		Comment:   req.Comment,
		STID:      st.ID,
		Scopes:    utils.SplitIgnoreEmpty(retScopes, " "),
		Audiences: retAudiences,
	}
	if err = db.Transact(func(tx *sqlx.Tx) error {
		if err = at.Store(tx); err != nil {
			return err
		}
		if err = eventService.LogEvent(tx, event.FromNumber(event.STEventATCreated, "Used grant_type super_token"), st.ID, networkData); err != nil {
			return err
		}
		if usedRestriction != nil {
			if err = usedRestriction.UsedAT(tx, st.ID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return model.ErrorToInternalServerErrorResponse(err)
	}
	return &model.Response{
		Status: fiber.StatusOK,
		Response: request.AccessTokenResponse{
			AccessToken: oidcRes.AccessToken,
			TokenType:   oidcRes.TokenType,
			ExpiresIn:   oidcRes.ExpiresIn,
			Scope:       retScopes,
			Audiences:   retAudiences,
		},
	}
}
