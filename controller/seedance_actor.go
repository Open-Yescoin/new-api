package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/doubao"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

type seedanceActorController struct {
	actors        *doubao.ActorService
	authorization *doubao.ActorAuthorizationService
	assets        *doubao.AssetService
}

func newDefaultSeedanceActorController() (*seedanceActorController, error) {
	httpClient, err := service.GetHttpClientWithProxy("")
	if err != nil {
		return nil, err
	}
	config := system_setting.VolcAssetConfig
	api := doubao.NewVolcAssetClient(config, httpClient)
	return &seedanceActorController{
		actors:        doubao.NewActorService(nil),
		authorization: doubao.NewActorAuthorizationService(api, config, nil),
		assets:        doubao.NewAssetService(api, doubao.NewModelAssetGroupRepository(), config),
	}, nil
}

func withDefaultSeedanceActorController(c *gin.Context, handle func(*seedanceActorController, *gin.Context)) {
	controller, err := newDefaultSeedanceActorController()
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	handle(controller, c)
}

func parseSeedanceActorId(c *gin.Context) (int, bool) {
	actorId, err := strconv.Atoi(strings.TrimSpace(c.Param("actor_id")))
	if err != nil || actorId <= 0 {
		respondSeedanceAssetError(c, doubao.ErrInvalidAssetRequest)
		return 0, false
	}
	return actorId, true
}

func (h *seedanceActorController) ListActors(c *gin.Context) {
	actors, err := h.actors.List(c.GetInt("id"))
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, actors)
}

func (h *seedanceActorController) CreateActor(c *gin.Context) {
	var request struct {
		DisplayName  string `json:"display_name"`
		DurationDays *int   `json:"duration_days"`
	}
	if !decodeSeedanceAssetRequest(c, &request) {
		return
	}
	actor, err := h.actors.Create(c.GetInt("id"), request.DisplayName, request.DurationDays)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, actor)
}

func (h *seedanceActorController) RenameActor(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	var request struct {
		DisplayName string `json:"display_name"`
	}
	if !decodeSeedanceAssetRequest(c, &request) {
		return
	}
	if err := h.actors.Rename(c.GetInt("id"), actorId, request.DisplayName); err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{})
}

func (h *seedanceActorController) DeleteActor(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	if err := h.actors.Delete(c.GetInt("id"), actorId); err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{})
}

func (h *seedanceActorController) AuthorizationStatus(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	actor, err := h.actors.Get(c.GetInt("id"), actorId)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	pending, err := model.HasPendingVolcAssetActorAuthorization(c.GetInt("id"), actorId, time.Now().Unix())
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{
		"configured": h.authorization.Configured(),
		"pending":    pending,
		"actor":      actor,
	})
}

func (h *seedanceActorController) CreateAuthorizationSession(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	session, err := h.authorization.CreateSession(c.Request.Context(), c.GetInt("id"), actorId)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, session)
}

func (h *seedanceActorController) AuthorizationResult(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	var request struct {
		BytedToken string `json:"byted_token"`
	}
	if !decodeSeedanceAssetRequest(c, &request) {
		return
	}
	result, err := h.authorization.Poll(c.Request.Context(), c.GetInt("id"), actorId, request.BytedToken)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, result)
}

func (h *seedanceActorController) RevokeActor(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	if err := h.actors.Revoke(c.GetInt("id"), actorId); err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{})
}

func (h *seedanceActorController) ListAssets(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	request, err := seedanceListAssetsRequest(c)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	response, err := h.assets.ListActorAssets(c.Request.Context(), c.GetInt("id"), actorId, request)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, newSeedanceAssetListResponse(response))
}

func (h *seedanceActorController) CreateAsset(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	var request struct {
		URL       string `json:"url"`
		AssetType string `json:"asset_type"`
	}
	if !decodeSeedanceAssetRequest(c, &request) {
		return
	}
	response, err := h.assets.CreateActorAsset(c.Request.Context(), c.GetInt("id"), actorId, doubao.CreateAssetRequest{
		URL: strings.TrimSpace(request.URL), AssetType: strings.TrimSpace(request.AssetType),
	})
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, response)
}

func (h *seedanceActorController) GetAsset(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	asset, err := h.assets.GetActorAsset(c.Request.Context(), c.GetInt("id"), actorId, doubao.GetAssetRequest{Id: strings.TrimSpace(c.Param("id"))})
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, newSeedanceAssetItemResponse(asset))
}

func (h *seedanceActorController) UpdateAsset(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	var request struct {
		Name string `json:"name"`
	}
	if !decodeSeedanceAssetRequest(c, &request) {
		return
	}
	err := h.assets.UpdateActorAsset(c.Request.Context(), c.GetInt("id"), actorId, doubao.UpdateAssetRequest{Id: strings.TrimSpace(c.Param("id")), Name: strings.TrimSpace(request.Name)})
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{})
}

func (h *seedanceActorController) DeleteAsset(c *gin.Context) {
	actorId, ok := parseSeedanceActorId(c)
	if !ok {
		return
	}
	err := h.assets.DeleteActorAsset(c.Request.Context(), c.GetInt("id"), actorId, doubao.DeleteAssetRequest{Id: strings.TrimSpace(c.Param("id"))})
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{})
}

func seedancePublicToken(c *gin.Context) (string, bool) {
	var request struct {
		Token string `json:"token"`
	}
	if !decodeSeedanceAssetRequest(c, &request) {
		return "", false
	}
	request.Token = strings.TrimSpace(request.Token)
	if request.Token == "" {
		respondSeedanceAssetError(c, doubao.ErrInvalidAssetRequest)
		return "", false
	}
	return request.Token, true
}

func (h *seedanceActorController) PublicInvitationDetails(c *gin.Context) {
	token, ok := seedancePublicToken(c)
	if !ok {
		return
	}
	details, err := h.authorization.InvitationDetails(token)
	if err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, details)
}

func (h *seedanceActorController) PublicAcceptConsent(c *gin.Context) {
	token, ok := seedancePublicToken(c)
	if !ok {
		return
	}
	if err := h.authorization.AcceptConsent(token); err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{})
}

func (h *seedanceActorController) PublicRevoke(c *gin.Context) {
	token, ok := seedancePublicToken(c)
	if !ok {
		return
	}
	if err := h.authorization.RevokeByToken(token); err != nil {
		respondSeedanceAssetError(c, err)
		return
	}
	respondSeedanceAssetSuccess(c, gin.H{})
}

func ListSeedanceActors(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).ListActors)
}
func CreateSeedanceActor(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).CreateActor)
}
func RenameSeedanceActor(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).RenameActor)
}
func DeleteSeedanceActor(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).DeleteActor)
}
func GetSeedanceActorAuthorization(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).AuthorizationStatus)
}
func CreateSeedanceActorAuthorizationSession(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).CreateAuthorizationSession)
}
func GetSeedanceActorAuthorizationResult(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).AuthorizationResult)
}
func RevokeSeedanceActor(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).RevokeActor)
}
func ListSeedanceActorAssets(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).ListAssets)
}
func CreateSeedanceActorAsset(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).CreateAsset)
}
func GetSeedanceActorAsset(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).GetAsset)
}
func UpdateSeedanceActorAsset(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).UpdateAsset)
}
func DeleteSeedanceActorAsset(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).DeleteAsset)
}
func GetSeedancePublicAuthorizationDetails(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).PublicInvitationDetails)
}
func AcceptSeedancePublicAuthorizationConsent(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).PublicAcceptConsent)
}
func RevokeSeedancePublicAuthorization(c *gin.Context) {
	withDefaultSeedanceActorController(c, (*seedanceActorController).PublicRevoke)
}
