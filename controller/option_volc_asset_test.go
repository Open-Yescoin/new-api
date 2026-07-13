package controller

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetOptionsRedactsVolcAssetSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldConfig := system_setting.VolcAssetConfig
	oldOptions := common.OptionMap
	system_setting.VolcAssetConfig = system_setting.VolcAssetSettings{
		AccessKey:   "ak-visible",
		SecretKey:   "must-not-leak",
		Region:      "ap-southeast-1",
		ProjectName: "default",
		GroupType:   "AIGC",
	}
	common.OptionMap = map[string]string{"VolcAssetConfig": `{"secret_key":"must-not-leak"}`}
	t.Cleanup(func() {
		system_setting.VolcAssetConfig = oldConfig
		common.OptionMap = oldOptions
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	GetOptions(c)

	require.Equal(t, 200, recorder.Code)
	require.False(t, strings.Contains(recorder.Body.String(), "must-not-leak"))
	var response struct {
		Data []*model.Option `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	var value string
	for _, option := range response.Data {
		if option.Key == "VolcAssetConfig" {
			value = option.Value
			break
		}
	}
	require.NotEmpty(t, value)
	var public system_setting.PublicVolcAssetSettings
	require.NoError(t, common.UnmarshalJsonStr(value, &public))
	require.True(t, public.SecretKeyConfigured)
	require.Equal(t, "ak-visible", public.AccessKey)
}
