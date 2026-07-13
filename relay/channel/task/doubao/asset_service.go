package doubao

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"gorm.io/gorm"
)

var (
	ErrInvalidAssetRequest     = errors.New("invalid Seedance asset request")
	ErrAssetGroupNotAuthorized = errors.New("real-person asset authorization is required")
	ErrAssetNotFound           = errors.New("asset not found")
)

type CreateVisualValidateSessionRequest struct {
	CallbackURL string `json:"CallbackURL"`
	ProjectName string `json:"ProjectName,omitempty"`
}

type CreateVisualValidateSessionResponse struct {
	BytedToken  string `json:"BytedToken"`
	H5Link      string `json:"H5Link"`
	CallbackURL string `json:"CallbackURL,omitempty"`
}

type GetVisualValidateResultRequest struct {
	BytedToken  string `json:"BytedToken"`
	ProjectName string `json:"ProjectName,omitempty"`
}

type GetVisualValidateResultResponse struct {
	GroupId string `json:"GroupId"`
}

type AssetFilter struct {
	GroupIds  []string `json:"GroupIds,omitempty"`
	GroupType string   `json:"GroupType,omitempty"`
	Statuses  []string `json:"Statuses,omitempty"`
	Name      string   `json:"Name,omitempty"`
}

type ListAssetsRequest struct {
	Filter      *AssetFilter `json:"Filter,omitempty"`
	PageNumber  int64        `json:"PageNumber,omitempty"`
	PageSize    int64        `json:"PageSize,omitempty"`
	SortBy      string       `json:"SortBy,omitempty"`
	SortOrder   string       `json:"SortOrder,omitempty"`
	ProjectName string       `json:"ProjectName,omitempty"`
}

type AssetError struct {
	Code    string `json:"Code,omitempty"`
	Message string `json:"Message,omitempty"`
}

type AssetItem struct {
	Id             string     `json:"Id"`
	Name           string     `json:"Name,omitempty"`
	URL            string     `json:"URL,omitempty"`
	GroupId        string     `json:"GroupId"`
	AssetType      string     `json:"AssetType,omitempty"`
	Status         string     `json:"Status,omitempty"`
	UpstreamStatus string     `json:"UpstreamStatus,omitempty"`
	Error          AssetError `json:"Error,omitempty"`
	ProjectName    string     `json:"ProjectName,omitempty"`
	CreateTime     string     `json:"CreateTime,omitempty"`
	UpdateTime     string     `json:"UpdateTime,omitempty"`
}

type ListAssetsResponse struct {
	Items      []AssetItem `json:"Items"`
	TotalCount int64       `json:"TotalCount"`
	PageNumber int64       `json:"PageNumber"`
	PageSize   int64       `json:"PageSize"`
}

type CreateAssetRequest struct {
	GroupId     string `json:"GroupId"`
	URL         string `json:"URL"`
	AssetType   string `json:"AssetType"`
	ProjectName string `json:"ProjectName,omitempty"`
}

type CreateAssetResponse struct {
	Id     string `json:"Id"`
	Status string `json:"Status,omitempty"`
}

type GetAssetRequest struct {
	Id          string `json:"Id"`
	ProjectName string `json:"ProjectName,omitempty"`
}

type UpdateAssetRequest struct {
	Id          string `json:"Id"`
	Name        string `json:"Name,omitempty"`
	ProjectName string `json:"ProjectName,omitempty"`
}

type DeleteAssetRequest struct {
	Id          string `json:"Id"`
	ProjectName string `json:"ProjectName,omitempty"`
}

type AssetGroupRepository interface {
	Get(userId int) (string, error)
	Save(userId int, groupId string) error
}

type modelAssetGroupRepository struct{}

func (modelAssetGroupRepository) Get(userId int) (string, error) {
	binding, err := model.GetVolcAssetUserGroup(userId)
	if err != nil {
		return "", err
	}
	return binding.GroupId, nil
}

func (modelAssetGroupRepository) Save(userId int, groupId string) error {
	return model.SaveVolcAssetUserGroup(userId, groupId)
}

type AssetService struct {
	api    AssetAPI
	groups AssetGroupRepository
	config system_setting.VolcAssetSettings
}

func NewAssetService(api AssetAPI, groups AssetGroupRepository, config system_setting.VolcAssetSettings) *AssetService {
	return &AssetService{api: api, groups: groups, config: config}
}

func (s *AssetService) CreateVisualValidateSession(ctx context.Context, userId int, request CreateVisualValidateSessionRequest) (*CreateVisualValidateSessionResponse, error) {
	if userId <= 0 || !isHTTPURL(request.CallbackURL) {
		return nil, fmt.Errorf("%w: a valid CallbackURL is required", ErrInvalidAssetRequest)
	}
	request.ProjectName = s.config.GetProjectName()
	var response CreateVisualValidateSessionResponse
	if err := s.api.Call(ctx, "CreateVisualValidateSession", request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *AssetService) GetVisualValidateResult(ctx context.Context, userId int, request GetVisualValidateResultRequest) (*GetVisualValidateResultResponse, error) {
	if userId <= 0 || strings.TrimSpace(request.BytedToken) == "" {
		return nil, fmt.Errorf("%w: BytedToken is required", ErrInvalidAssetRequest)
	}
	request.ProjectName = s.config.GetProjectName()
	var response GetVisualValidateResultResponse
	if err := s.api.Call(ctx, "GetVisualValidateResult", request, &response); err != nil {
		return nil, err
	}
	if response.GroupId == "" {
		return nil, fmt.Errorf("%w: verification did not return GroupId", ErrAssetGroupNotAuthorized)
	}
	if err := s.groups.Save(userId, response.GroupId); err != nil {
		return nil, fmt.Errorf("save verified asset group: %w", err)
	}
	return &response, nil
}

func (s *AssetService) ListAssets(ctx context.Context, userId int, request ListAssetsRequest) (*ListAssetsResponse, error) {
	groupId, err := s.authorizedGroup(userId)
	if err != nil {
		return nil, err
	}
	if request.Filter == nil {
		request.Filter = &AssetFilter{}
	}
	request.Filter.GroupIds = []string{groupId}
	request.Filter.GroupType = s.config.GetGroupType()
	request.ProjectName = s.config.GetProjectName()
	if request.PageNumber <= 0 {
		request.PageNumber = 1
	}
	if request.PageSize <= 0 {
		request.PageSize = 20
	}
	var response ListAssetsResponse
	if err := s.api.Call(ctx, "ListAssets", request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *AssetService) CreateAsset(ctx context.Context, userId int, request CreateAssetRequest) (*CreateAssetResponse, error) {
	if !isHTTPURL(request.URL) {
		return nil, fmt.Errorf("%w: URL must be an absolute HTTP(S) URL", ErrInvalidAssetRequest)
	}
	if request.AssetType != "Image" && request.AssetType != "Video" {
		return nil, fmt.Errorf("%w: AssetType must be Image or Video", ErrInvalidAssetRequest)
	}
	groupId, err := s.authorizedGroup(userId)
	if err != nil {
		return nil, err
	}
	request.GroupId = groupId
	request.ProjectName = s.config.GetProjectName()
	var response CreateAssetResponse
	if err := s.api.Call(ctx, "CreateAsset", request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *AssetService) GetAsset(ctx context.Context, userId int, request GetAssetRequest) (*AssetItem, error) {
	if strings.TrimSpace(request.Id) == "" {
		return nil, fmt.Errorf("%w: Id is required", ErrInvalidAssetRequest)
	}
	groupId, err := s.authorizedGroup(userId)
	if err != nil {
		return nil, err
	}
	return s.getOwnedAsset(ctx, request.Id, groupId)
}

func (s *AssetService) UpdateAsset(ctx context.Context, userId int, request UpdateAssetRequest) error {
	if strings.TrimSpace(request.Id) == "" {
		return fmt.Errorf("%w: Id is required", ErrInvalidAssetRequest)
	}
	groupId, err := s.authorizedGroup(userId)
	if err != nil {
		return err
	}
	if _, err := s.getOwnedAsset(ctx, request.Id, groupId); err != nil {
		return err
	}
	request.ProjectName = s.config.GetProjectName()
	return s.api.Call(ctx, "UpdateAsset", request, &struct{}{})
}

func (s *AssetService) DeleteAsset(ctx context.Context, userId int, request DeleteAssetRequest) error {
	if strings.TrimSpace(request.Id) == "" {
		return fmt.Errorf("%w: Id is required", ErrInvalidAssetRequest)
	}
	groupId, err := s.authorizedGroup(userId)
	if err != nil {
		return err
	}
	if _, err := s.getOwnedAsset(ctx, request.Id, groupId); err != nil {
		return err
	}
	request.ProjectName = s.config.GetProjectName()
	return s.api.Call(ctx, "DeleteAsset", request, &struct{}{})
}

func (s *AssetService) authorizedGroup(userId int) (string, error) {
	if userId <= 0 {
		return "", fmt.Errorf("%w: invalid user", ErrInvalidAssetRequest)
	}
	groupId, err := s.groups.Get(userId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrAssetGroupNotAuthorized
	}
	if err != nil {
		return "", fmt.Errorf("load verified asset group: %w", err)
	}
	if groupId == "" {
		return "", ErrAssetGroupNotAuthorized
	}
	return groupId, nil
}

func (s *AssetService) getOwnedAsset(ctx context.Context, assetId, groupId string) (*AssetItem, error) {
	request := GetAssetRequest{Id: assetId, ProjectName: s.config.GetProjectName()}
	var asset AssetItem
	if err := s.api.Call(ctx, "GetAsset", request, &asset); err != nil {
		return nil, err
	}
	if asset.Id == "" || asset.GroupId != groupId {
		return nil, ErrAssetNotFound
	}
	return &asset, nil
}

func isHTTPURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
