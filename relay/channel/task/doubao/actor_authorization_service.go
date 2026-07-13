package doubao

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const seedanceActorConsentVersion = "2026-07-13"

type ActorAuthorizationService struct {
	api    AssetAPI
	config system_setting.VolcAssetSettings
	now    func() time.Time
}

type ActorAuthorizationSession struct {
	BytedToken      string `json:"byted_token"`
	H5Link          string `json:"h5_link"`
	InvitationToken string `json:"invitation_token"`
	RevocationToken string `json:"revocation_token"`
	ExpiresAt       int64  `json:"expires_at"`
}

type ActorAuthorizationInvitationDetails struct {
	ActorId        int    `json:"actor_id"`
	ActorName      string `json:"actor_name"`
	DurationDays   *int   `json:"duration_days"`
	ConsentVersion string `json:"consent_version"`
	ExpiresAt      int64  `json:"expires_at"`
}

func NewActorAuthorizationService(api AssetAPI, config system_setting.VolcAssetSettings, now func() time.Time) *ActorAuthorizationService {
	if now == nil {
		now = time.Now
	}
	return &ActorAuthorizationService{api: api, config: config, now: now}
}

func (s *ActorAuthorizationService) Configured() bool {
	_, err := s.authorizationCallbackURL()
	return err == nil
}

func (s *ActorAuthorizationService) CreateSession(ctx context.Context, userId, actorId int) (*ActorAuthorizationSession, error) {
	actor, err := model.GetVolcAssetActorForUser(userId, actorId)
	if err != nil {
		return nil, err
	}
	callbackURL, err := s.authorizationCallbackURL()
	if err != nil {
		return nil, err
	}
	var upstream CreateVisualValidateSessionResponse
	if err := s.api.Call(ctx, "CreateVisualValidateSession", CreateVisualValidateSessionRequest{
		CallbackURL: callbackURL,
		ProjectName: s.config.GetProjectName(),
	}, &upstream); err != nil {
		return nil, err
	}
	rawToken := strings.TrimSpace(upstream.BytedToken)
	h5Link := strings.TrimSpace(upstream.H5Link)
	if rawToken == "" || h5Link == "" || !isHTTPSURL(h5Link) {
		return nil, ErrInvalidAssetAuthorizationResponse
	}
	invitationToken, err := common.GenerateRandomKey(48)
	if err != nil {
		return nil, fmt.Errorf("generate Seedance invitation token: %w", err)
	}
	revocationToken, err := common.GenerateRandomKey(48)
	if err != nil {
		return nil, fmt.Errorf("generate Seedance revocation token: %w", err)
	}
	now := s.now().Unix()
	expiresAt := now + int64(assetAuthorizationSessionTTL/time.Second)
	_, err = model.StartVolcAssetActorAuthorization(
		userId,
		actorId,
		authorizationTokenDigest(rawToken),
		authorizationTokenDigest(invitationToken),
		authorizationTokenDigest(revocationToken),
		actor.AuthorizationDurationDays,
		seedanceActorConsentVersion,
		expiresAt,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("save Seedance actor authorization session: %w", err)
	}
	return &ActorAuthorizationSession{
		BytedToken: rawToken, H5Link: h5Link, InvitationToken: invitationToken,
		RevocationToken: revocationToken, ExpiresAt: expiresAt,
	}, nil
}

func (s *ActorAuthorizationService) InvitationDetails(invitationToken string) (*ActorAuthorizationInvitationDetails, error) {
	invitationToken = strings.TrimSpace(invitationToken)
	if invitationToken == "" {
		return nil, model.ErrVolcAssetAuthorizationNotFound
	}
	invitation, err := model.GetVolcAssetAuthorizationInvitation(authorizationTokenDigest(invitationToken), s.now().Unix())
	if err != nil {
		return nil, err
	}
	return &ActorAuthorizationInvitationDetails{
		ActorId: invitation.Actor.Id, ActorName: invitation.Actor.DisplayName,
		DurationDays: invitation.Session.DurationDays, ConsentVersion: invitation.Session.ConsentVersion,
		ExpiresAt: invitation.Session.ExpiresAt,
	}, nil
}

func (s *ActorAuthorizationService) AcceptConsent(invitationToken string) error {
	invitationToken = strings.TrimSpace(invitationToken)
	if invitationToken == "" {
		return model.ErrVolcAssetAuthorizationNotFound
	}
	_, err := model.AcceptVolcAssetAuthorizationConsent(authorizationTokenDigest(invitationToken), s.now().Unix())
	return err
}

func (s *ActorAuthorizationService) Poll(ctx context.Context, userId, actorId int, rawToken string) (*AuthorizationResult, error) {
	rawToken = strings.TrimSpace(rawToken)
	if userId <= 0 || actorId <= 0 || rawToken == "" {
		return nil, fmt.Errorf("%w: byted_token is required", ErrInvalidAssetRequest)
	}
	now := s.now().Unix()
	digest := authorizationTokenDigest(rawToken)
	if _, err := model.FindPendingVolcAssetActorAuthorization(userId, actorId, digest, now); err != nil {
		if errors.Is(err, model.ErrVolcAssetAuthorizationExpired) {
			return nil, ErrAssetAuthorizationExpired
		}
		return nil, err
	}
	var upstream GetVisualValidateResultResponse
	if err := s.api.Call(ctx, "GetVisualValidateResult", GetVisualValidateResultRequest{
		BytedToken: rawToken, ProjectName: s.config.GetProjectName(),
	}, &upstream); err != nil {
		var apiErr *AssetAPIError
		if errors.As(err, &apiErr) && apiErr.Code == "40004" {
			_ = model.ExpireVolcAssetActorAuthorization(userId, actorId, digest, now)
			return nil, ErrAssetAuthorizationExpired
		}
		return nil, err
	}
	groupId := strings.TrimSpace(upstream.GroupId)
	if groupId == "" {
		return &AuthorizationResult{Status: "pending", Authorized: false}, nil
	}
	if err := model.CompleteVolcAssetActorAuthorization(userId, actorId, digest, groupId, now); err != nil {
		return nil, err
	}
	return &AuthorizationResult{Status: "completed", Authorized: true}, nil
}

func (s *ActorAuthorizationService) RevokeByToken(revocationToken string) error {
	revocationToken = strings.TrimSpace(revocationToken)
	if revocationToken == "" {
		return model.ErrVolcAssetAuthorizationNotFound
	}
	return model.RevokeVolcAssetActorByToken(authorizationTokenDigest(revocationToken), s.now().Unix())
}

func (s *ActorAuthorizationService) authorizationCallbackURL() (string, error) {
	if strings.TrimSpace(s.config.AccessKey) == "" || strings.TrimSpace(s.config.SecretKey) == "" {
		return "", ErrVolcAssetNotConfigured
	}
	callbackURL, err := s.config.GetAuthorizationCallbackURL()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrVolcAssetNotConfigured, err)
	}
	return callbackURL, nil
}
