package doubao

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

type AssetHandler struct {
	service *AssetService
}

func NewAssetHandler(assetService *AssetService) *AssetHandler {
	return &AssetHandler{service: assetService}
}

func (h *AssetHandler) CreateVisualValidateSession(c *gin.Context) {
	request, ok := decodeAssetRequest[CreateVisualValidateSessionRequest](c)
	if !ok {
		return
	}
	response, err := h.service.CreateVisualValidateSession(c.Request.Context(), c.GetInt("id"), request)
	if err != nil {
		respondAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *AssetHandler) GetVisualValidateResult(c *gin.Context) {
	request, ok := decodeAssetRequest[GetVisualValidateResultRequest](c)
	if !ok {
		return
	}
	response, err := h.service.GetVisualValidateResult(c.Request.Context(), c.GetInt("id"), request)
	if err != nil {
		respondAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *AssetHandler) ListAssets(c *gin.Context) {
	request, ok := decodeAssetRequest[ListAssetsRequest](c)
	if !ok {
		return
	}
	response, err := h.service.ListAssets(c.Request.Context(), c.GetInt("id"), request)
	if err != nil {
		respondAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *AssetHandler) GetAsset(c *gin.Context) {
	request, ok := decodeAssetRequest[GetAssetRequest](c)
	if !ok {
		return
	}
	response, err := h.service.GetAsset(c.Request.Context(), c.GetInt("id"), request)
	if err != nil {
		respondAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *AssetHandler) CreateAsset(c *gin.Context) {
	request, ok := decodeAssetRequest[CreateAssetRequest](c)
	if !ok {
		return
	}
	response, err := h.service.CreateAsset(c.Request.Context(), c.GetInt("id"), request)
	if err != nil {
		respondAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *AssetHandler) UpdateAsset(c *gin.Context) {
	request, ok := decodeAssetRequest[UpdateAssetRequest](c)
	if !ok {
		return
	}
	if err := h.service.UpdateAsset(c.Request.Context(), c.GetInt("id"), request); err != nil {
		respondAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *AssetHandler) DeleteAsset(c *gin.Context) {
	request, ok := decodeAssetRequest[DeleteAssetRequest](c)
	if !ok {
		return
	}
	if err := h.service.DeleteAsset(c.Request.Context(), c.GetInt("id"), request); err != nil {
		respondAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func decodeAssetRequest[T any](c *gin.Context) (T, bool) {
	var request T
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "invalid_request", "message": "invalid JSON request body"},
		})
		return request, false
	}
	return request, true
}

func respondAssetError(c *gin.Context, err error) {
	statusCode := http.StatusBadGateway
	code := "asset_upstream_error"
	message := "Seedance asset library request failed"

	switch {
	case errors.Is(err, ErrInvalidAssetRequest):
		statusCode = http.StatusBadRequest
		code = "invalid_request"
		message = err.Error()
	case errors.Is(err, ErrAssetGroupNotAuthorized):
		statusCode = http.StatusConflict
		code = "asset_authorization_required"
		message = ErrAssetGroupNotAuthorized.Error()
	case errors.Is(err, ErrAssetNotFound):
		statusCode = http.StatusNotFound
		code = "asset_not_found"
		message = ErrAssetNotFound.Error()
	case errors.Is(err, ErrVolcAssetNotConfigured):
		statusCode = http.StatusServiceUnavailable
		code = "asset_library_not_configured"
		message = ErrVolcAssetNotConfigured.Error()
	default:
		var apiErr *AssetAPIError
		if errors.As(err, &apiErr) {
			if apiErr.StatusCode >= 400 && apiErr.StatusCode <= 599 {
				statusCode = apiErr.StatusCode
			}
			code = apiErr.Code
			message = apiErr.Message
		}
	}

	c.JSON(statusCode, gin.H{
		"error": gin.H{"code": code, "message": message},
	})
}

func newDefaultAssetHandler() (*AssetHandler, error) {
	httpClient, err := service.GetHttpClientWithProxy("")
	if err != nil {
		return nil, err
	}
	config := system_setting.VolcAssetConfig
	api := NewVolcAssetClient(config, httpClient)
	assetService := NewAssetService(api, modelAssetGroupRepository{}, config)
	return NewAssetHandler(assetService), nil
}

func withDefaultAssetHandler(c *gin.Context, handle func(*AssetHandler, *gin.Context)) {
	handler, err := newDefaultAssetHandler()
	if err != nil {
		respondAssetError(c, err)
		return
	}
	handle(handler, c)
}

func HandleCreateVisualValidateSession(c *gin.Context) {
	withDefaultAssetHandler(c, (*AssetHandler).CreateVisualValidateSession)
}

func HandleGetVisualValidateResult(c *gin.Context) {
	withDefaultAssetHandler(c, (*AssetHandler).GetVisualValidateResult)
}

func HandleListAssets(c *gin.Context) {
	withDefaultAssetHandler(c, (*AssetHandler).ListAssets)
}

func HandleGetAsset(c *gin.Context) {
	withDefaultAssetHandler(c, (*AssetHandler).GetAsset)
}

func HandleCreateAsset(c *gin.Context) {
	withDefaultAssetHandler(c, (*AssetHandler).CreateAsset)
}

func HandleUpdateAsset(c *gin.Context) {
	withDefaultAssetHandler(c, (*AssetHandler).UpdateAsset)
}

func HandleDeleteAsset(c *gin.Context) {
	withDefaultAssetHandler(c, (*AssetHandler).DeleteAsset)
}
