package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/doubao"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

type seedanceAuthorizationService interface {
	Status(userId int) (*doubao.AuthorizationStatus, error)
	CreateSession(ctx context.Context, userId int) (*doubao.AuthorizationSession, error)
	Poll(ctx context.Context, userId int, bytedToken string) (*doubao.AuthorizationResult, error)
}

type seedanceAssetService interface {
	ListAssets(ctx context.Context, userId int, request doubao.ListAssetsRequest) (*doubao.ListAssetsResponse, error)
	CreateAsset(ctx context.Context, userId int, request doubao.CreateAssetRequest) (*doubao.CreateAssetResponse, error)
	GetAsset(ctx context.Context, userId int, request doubao.GetAssetRequest) (*doubao.AssetItem, error)
	UpdateAsset(ctx context.Context, userId int, request doubao.UpdateAssetRequest) error
	DeleteAsset(ctx context.Context, userId int, request doubao.DeleteAssetRequest) error
}

type seedanceAssetController struct {
	authorization seedanceAuthorizationService
	assets        seedanceAssetService
}

func newSeedanceAssetController(authorization seedanceAuthorizationService, assets seedanceAssetService) *seedanceAssetController {
	return &seedanceAssetController{authorization: authorization, assets: assets}
}

func (h *seedanceAssetController) GetAuthorizationStatus(c *gin.Context) {
	status, err := h.authorization.Status(c.GetInt("id"))
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, status)
}

func (h *seedanceAssetController) CreateAuthorizationSession(c *gin.Context) {
	session, err := h.authorization.CreateSession(c.Request.Context(), c.GetInt("id"))
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, session)
}

func (h *seedanceAssetController) GetAuthorizationResult(c *gin.Context) {
	var request struct {
		BytedToken string `json:"byted_token"`
	}
	if !decodeSeedanceAssetRequest(c, &request) {
		return
	}
	request.BytedToken = strings.TrimSpace(request.BytedToken)
	if request.BytedToken == "" {
		respondSeedanceAssetError(c, doubao.ErrInvalidAssetRequest)
		return
	}
	result, err := h.authorization.Poll(c.Request.Context(), c.GetInt("id"), request.BytedToken)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, result)
}

func (h *seedanceAssetController) ListAssets(c *gin.Context) {
	request, err := seedanceListAssetsRequest(c)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	response, err := h.assets.ListAssets(c.Request.Context(), c.GetInt("id"), request)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, response)
}

func (h *seedanceAssetController) CreateAsset(c *gin.Context) {
	var request struct {
		URL       string `json:"url"`
		AssetType string `json:"asset_type"`
	}
	if !decodeSeedanceAssetRequest(c, &request) {
		return
	}
	response, err := h.assets.CreateAsset(c.Request.Context(), c.GetInt("id"), doubao.CreateAssetRequest{
		URL:       strings.TrimSpace(request.URL),
		AssetType: strings.TrimSpace(request.AssetType),
	})
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, response)
}

func (h *seedanceAssetController) GetAsset(c *gin.Context) {
	response, err := h.assets.GetAsset(c.Request.Context(), c.GetInt("id"), doubao.GetAssetRequest{
		Id: strings.TrimSpace(c.Param("id")),
	})
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, response)
}

func (h *seedanceAssetController) UpdateAsset(c *gin.Context) {
	var request struct {
		Name string `json:"name"`
	}
	if !decodeSeedanceAssetRequest(c, &request) {
		return
	}
	err := h.assets.UpdateAsset(c.Request.Context(), c.GetInt("id"), doubao.UpdateAssetRequest{
		Id:   strings.TrimSpace(c.Param("id")),
		Name: strings.TrimSpace(request.Name),
	})
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{})
}

func (h *seedanceAssetController) DeleteAsset(c *gin.Context) {
	err := h.assets.DeleteAsset(c.Request.Context(), c.GetInt("id"), doubao.DeleteAssetRequest{
		Id: strings.TrimSpace(c.Param("id")),
	})
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{})
}

func seedanceListAssetsRequest(c *gin.Context) (doubao.ListAssetsRequest, error) {
	pageNumber, err := parseOptionalPositiveInt64(c.Query("page_number"))
	if err != nil {
		return doubao.ListAssetsRequest{}, err
	}
	pageSize, err := parseOptionalPositiveInt64(c.Query("page_size"))
	if err != nil {
		return doubao.ListAssetsRequest{}, err
	}
	filter := &doubao.AssetFilter{
		Name: strings.TrimSpace(c.Query("name")),
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		filter.Statuses = []string{status}
	}
	return doubao.ListAssetsRequest{
		Filter:     filter,
		PageNumber: pageNumber,
		PageSize:   pageSize,
		SortBy:     strings.TrimSpace(c.Query("sort_by")),
		SortOrder:  strings.TrimSpace(c.Query("sort_order")),
	}, nil
}

func parseOptionalPositiveInt64(raw string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, doubao.ErrInvalidAssetRequest
	}
	return value, nil
}

func decodeSeedanceAssetRequest(c *gin.Context, target any) bool {
	if err := common.DecodeJson(c.Request.Body, target); err != nil {
		respondSeedanceAssetError(c, doubao.ErrInvalidAssetRequest)
		return false
	}
	return true
}

func respondSeedanceAssetSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func respondSeedanceAssetError(c *gin.Context, err error) {
	statusCode := http.StatusBadGateway
	code := "asset_upstream_error"
	message := "Seedance asset library request failed"

	switch {
	case errors.Is(err, doubao.ErrInvalidAssetRequest),
		errors.Is(err, model.ErrVolcAssetAuthorizationNotFound):
		statusCode = http.StatusBadRequest
		code = "invalid_request"
		message = "Invalid Seedance asset request"
	case errors.Is(err, doubao.ErrAssetAuthorizationExpired),
		errors.Is(err, model.ErrVolcAssetAuthorizationExpired):
		statusCode = http.StatusGone
		code = "authorization_expired"
		message = doubao.ErrAssetAuthorizationExpired.Error()
	case errors.Is(err, doubao.ErrAssetGroupNotAuthorized):
		statusCode = http.StatusConflict
		code = "asset_authorization_required"
		message = doubao.ErrAssetGroupNotAuthorized.Error()
	case errors.Is(err, doubao.ErrAssetNotFound):
		statusCode = http.StatusNotFound
		code = "asset_not_found"
		message = doubao.ErrAssetNotFound.Error()
	case errors.Is(err, doubao.ErrVolcAssetNotConfigured):
		statusCode = http.StatusServiceUnavailable
		code = "asset_library_not_configured"
		message = "Seedance asset library is not configured"
	default:
		var apiErr *doubao.AssetAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= http.StatusInternalServerError {
			statusCode = http.StatusServiceUnavailable
		}
	}

	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   gin.H{"code": code, "message": message},
	})
}

func newDefaultSeedanceAssetController() (*seedanceAssetController, error) {
	httpClient, err := service.GetHttpClientWithProxy("")
	if err != nil {
		return nil, err
	}
	config := system_setting.VolcAssetConfig
	api := doubao.NewVolcAssetClient(config, httpClient)
	groups := doubao.NewModelAssetGroupRepository()
	authorization := doubao.NewAuthorizationService(
		api,
		groups,
		doubao.NewModelAssetAuthorizationRepository(),
		config,
		nil,
	)
	assets := doubao.NewAssetService(api, groups, config)
	return newSeedanceAssetController(authorization, assets), nil
}

func withDefaultSeedanceAssetController(c *gin.Context, handle func(*seedanceAssetController, *gin.Context)) {
	handler, err := newDefaultSeedanceAssetController()
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	handle(handler, c)
}

func GetSeedanceAuthorizationStatus(c *gin.Context) {
	withDefaultSeedanceAssetController(c, (*seedanceAssetController).GetAuthorizationStatus)
}

func CreateSeedanceAuthorizationSession(c *gin.Context) {
	withDefaultSeedanceAssetController(c, (*seedanceAssetController).CreateAuthorizationSession)
}

func GetSeedanceAuthorizationResult(c *gin.Context) {
	withDefaultSeedanceAssetController(c, (*seedanceAssetController).GetAuthorizationResult)
}

func ListSeedanceAssets(c *gin.Context) {
	withDefaultSeedanceAssetController(c, (*seedanceAssetController).ListAssets)
}

func CreateSeedanceAsset(c *gin.Context) {
	withDefaultSeedanceAssetController(c, (*seedanceAssetController).CreateAsset)
}

func GetSeedanceAsset(c *gin.Context) {
	withDefaultSeedanceAssetController(c, (*seedanceAssetController).GetAsset)
}

func UpdateSeedanceAsset(c *gin.Context) {
	withDefaultSeedanceAssetController(c, (*seedanceAssetController).UpdateAsset)
}

func DeleteSeedanceAsset(c *gin.Context) {
	withDefaultSeedanceAssetController(c, (*seedanceAssetController).DeleteAsset)
}
