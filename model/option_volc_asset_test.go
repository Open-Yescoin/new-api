package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func TestPrepareOptionValueMergesVolcAssetSecret(t *testing.T) {
	old := system_setting.VolcAssetConfig
	system_setting.VolcAssetConfig = system_setting.VolcAssetSettings{SecretKey: "existing-secret"}
	t.Cleanup(func() { system_setting.VolcAssetConfig = old })

	incoming, err := common.Marshal(system_setting.VolcAssetSettings{
		AccessKey:   "new-ak",
		Region:      "ap-southeast-1",
		ProjectName: "default",
		GroupType:   "AIGC",
	})
	require.NoError(t, err)

	prepared, err := prepareOptionValue("VolcAssetConfig", string(incoming))
	require.NoError(t, err)

	var stored system_setting.VolcAssetSettings
	require.NoError(t, common.UnmarshalJsonStr(prepared, &stored))
	require.Equal(t, "existing-secret", stored.SecretKey)
	require.Equal(t, "new-ak", stored.AccessKey)
}
