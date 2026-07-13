package doubao

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"gorm.io/gorm"
)

const assetAuthorizationSessionTTL = 15 * time.Minute

var (
	ErrInvalidAssetAuthorizationResponse = errors.New("BytePlus returned an invalid asset authorization response")
	ErrAssetAuthorizationExpired         = errors.New("real-person asset authorization has expired")
)

type AuthorizationStatus struct {
	Configured bool  `json:"configured"`
	Authorized bool  `json:"authorized"`
	Pending    bool  `json:"pending"`
	UpdatedAt  int64 `json:"updated_at,omitempty"`
}

type AuthorizationSession struct {
	BytedToken string `json:"byted_token"`
	H5Link     string `json:"h5_link"`
	ExpiresAt  int64  `json:"expires_at"`
}

type AuthorizationResult struct {
	Status     string `json:"status"`
	Authorized bool   `json:"authorized"`
}

type AssetAuthorizationRepository interface {
	Start(userId int, tokenHash string, expiresAt, now int64) error
	FindPending(userId int, tokenHash string, now int64) error
	HasPending(userId int, now int64) (bool, error)
	Expire(userId int, tokenHash string, now int64) error
	Complete(userId int, tokenHash, groupId string, now int64) error
}

type modelAssetAuthorizationRepository struct{}

func NewModelAssetAuthorizationRepository() AssetAuthorizationRepository {
	return modelAssetAuthorizationRepository{}
}

func (modelAssetAuthorizationRepository) Start(userId int, tokenHash string, expiresAt, now int64) error {
	_, err := model.StartVolcAssetAuthorization(userId, tokenHash, expiresAt, now)
	return err
}

func (modelAssetAuthorizationRepository) FindPending(userId int, tokenHash string, now int64) error {
	_, err := model.FindPendingVolcAssetAuthorization(userId, tokenHash, now)
	return err
}

func (modelAssetAuthorizationRepository) HasPending(userId int, now int64) (bool, error) {
	return model.HasPendingVolcAssetAuthorization(userId, now)
}

func (modelAssetAuthorizationRepository) Expire(userId int, tokenHash string, now int64) error {
	return model.ExpireVolcAssetAuthorization(userId, tokenHash, now)
}

func (modelAssetAuthorizationRepository) Complete(userId int, tokenHash, groupId string, now int64) error {
	return model.CompleteVolcAssetAuthorization(userId, tokenHash, groupId, now)
}

type AuthorizationService struct {
	api      AssetAPI
	groups   AssetGroupRepository
	sessions AssetAuthorizationRepository
	config   system_setting.VolcAssetSettings
	now      func() time.Time
}

func NewAuthorizationService(
	api AssetAPI,
	groups AssetGroupRepository,
	sessions AssetAuthorizationRepository,
	config system_setting.VolcAssetSettings,
	now func() time.Time,
) *AuthorizationService {
	if now == nil {
		now = time.Now
	}
	return &AuthorizationService{
		api:      api,
		groups:   groups,
		sessions: sessions,
		config:   config,
		now:      now,
	}
}

func (s *AuthorizationService) Status(userId int) (*AuthorizationStatus, error) {
	if userId <= 0 {
		return nil, fmt.Errorf("%w: invalid user", ErrInvalidAssetRequest)
	}
	status := &AuthorizationStatus{Configured: s.configured()}
	binding, err := s.groups.Get(userId)
	if err == nil {
		status.Authorized = strings.TrimSpace(binding.GroupId) != ""
		status.UpdatedAt = binding.UpdatedAt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load verified asset group: %w", err)
	}
	status.Pending, err = s.sessions.HasPending(userId, s.now().Unix())
	if err != nil {
		return nil, fmt.Errorf("load asset authorization session: %w", err)
	}
	return status, nil
}

func (s *AuthorizationService) CreateSession(ctx context.Context, userId int) (*AuthorizationSession, error) {
	if userId <= 0 {
		return nil, fmt.Errorf("%w: invalid user", ErrInvalidAssetRequest)
	}
	callbackURL, err := s.authorizationCallbackURL()
	if err != nil {
		return nil, err
	}

	request := CreateVisualValidateSessionRequest{
		CallbackURL: callbackURL,
		ProjectName: s.config.GetProjectName(),
	}
	var upstream CreateVisualValidateSessionResponse
	if err := s.api.Call(ctx, "CreateVisualValidateSession", request, &upstream); err != nil {
		return nil, err
	}
	rawToken := strings.TrimSpace(upstream.BytedToken)
	h5Link := strings.TrimSpace(upstream.H5Link)
	if rawToken == "" || h5Link == "" || !isHTTPSURL(h5Link) {
		return nil, ErrInvalidAssetAuthorizationResponse
	}

	now := s.now().Unix()
	expiresAt := now + int64(assetAuthorizationSessionTTL/time.Second)
	digest := authorizationTokenDigest(rawToken)
	if err := s.sessions.Start(userId, digest, expiresAt, now); err != nil {
		return nil, fmt.Errorf("save asset authorization session: %w", err)
	}
	return &AuthorizationSession{
		BytedToken: rawToken,
		H5Link:     h5Link,
		ExpiresAt:  expiresAt,
	}, nil
}

func (s *AuthorizationService) Poll(ctx context.Context, userId int, rawToken string) (*AuthorizationResult, error) {
	rawToken = strings.TrimSpace(rawToken)
	if userId <= 0 || rawToken == "" {
		return nil, fmt.Errorf("%w: byted_token is required", ErrInvalidAssetRequest)
	}
	now := s.now().Unix()
	digest := authorizationTokenDigest(rawToken)
	if err := s.sessions.FindPending(userId, digest, now); err != nil {
		if errors.Is(err, model.ErrVolcAssetAuthorizationExpired) {
			return nil, ErrAssetAuthorizationExpired
		}
		return nil, err
	}

	request := GetVisualValidateResultRequest{
		BytedToken:  rawToken,
		ProjectName: s.config.GetProjectName(),
	}
	var upstream GetVisualValidateResultResponse
	if err := s.api.Call(ctx, "GetVisualValidateResult", request, &upstream); err != nil {
		var upstreamErr *AssetAPIError
		if errors.As(err, &upstreamErr) && upstreamErr.Code == "40004" {
			if expireErr := s.sessions.Expire(userId, digest, now); expireErr != nil {
				return nil, fmt.Errorf("expire asset authorization session: %w", expireErr)
			}
			return nil, ErrAssetAuthorizationExpired
		}
		return nil, err
	}
	groupId := strings.TrimSpace(upstream.GroupId)
	if groupId == "" {
		return &AuthorizationResult{Status: "pending", Authorized: false}, nil
	}
	if err := s.sessions.Complete(userId, digest, groupId, now); err != nil {
		if errors.Is(err, model.ErrVolcAssetAuthorizationExpired) {
			return nil, ErrAssetAuthorizationExpired
		}
		return nil, fmt.Errorf("complete asset authorization session: %w", err)
	}
	return &AuthorizationResult{Status: "completed", Authorized: true}, nil
}

func (s *AuthorizationService) configured() bool {
	if strings.TrimSpace(s.config.AccessKey) == "" || strings.TrimSpace(s.config.SecretKey) == "" {
		return false
	}
	_, err := s.config.GetAuthorizationCallbackURL()
	return err == nil
}

func (s *AuthorizationService) authorizationCallbackURL() (string, error) {
	if strings.TrimSpace(s.config.AccessKey) == "" || strings.TrimSpace(s.config.SecretKey) == "" {
		return "", ErrVolcAssetNotConfigured
	}
	callbackURL, err := s.config.GetAuthorizationCallbackURL()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrVolcAssetNotConfigured, err)
	}
	return callbackURL, nil
}

func authorizationTokenDigest(rawToken string) string {
	return hex.EncodeToString(common.Sha256Raw([]byte(strings.TrimSpace(rawToken))))
}
