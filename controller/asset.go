package controller

import (
	"github.com/QuantumNous/new-api/relay/channel/task/doubao"
	"github.com/gin-gonic/gin"
)

func RelayCreateVisualValidateSession(c *gin.Context) {
	doubao.HandleCreateVisualValidateSession(c)
}

func RelayGetVisualValidateResult(c *gin.Context) {
	doubao.HandleGetVisualValidateResult(c)
}

func RelayListAssets(c *gin.Context) {
	doubao.HandleListAssets(c)
}

func RelayGetAsset(c *gin.Context) {
	doubao.HandleGetAsset(c)
}

func RelayCreateAsset(c *gin.Context) {
	doubao.HandleCreateAsset(c)
}

func RelayUpdateAsset(c *gin.Context) {
	doubao.HandleUpdateAsset(c)
}

func RelayDeleteAsset(c *gin.Context) {
	doubao.HandleDeleteAsset(c)
}
