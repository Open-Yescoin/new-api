package system_setting

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestVolcAssetSettingsDefaults(t *testing.T) {
	cfg := VolcAssetSettings{}

	require.Equal(t, "ap-southeast-1", cfg.GetRegion())
	require.Equal(t, "default", cfg.GetProjectName())
	require.Equal(t, "LivenessFace", cfg.GetGroupType())
	require.Equal(t, "https://ark.ap-southeast-1.byteplusapi.com", cfg.GetBaseURL())
}

func TestMergeVolcAssetSettingsPreservesSecret(t *testing.T) {
	current := VolcAssetSettings{
		AccessKey:   "old-ak",
		SecretKey:   "existing-secret",
		Region:      "ap-southeast-1",
		ProjectName: "default",
		GroupType:   "AIGC",
	}
	incoming := VolcAssetSettings{
		AccessKey:   "new-ak",
		Region:      "ap-southeast-1",
		ProjectName: "project-a",
		GroupType:   "AIGC",
	}

	got := MergeVolcAssetSettings(current, incoming)

	require.Equal(t, "new-ak", got.AccessKey)
	require.Equal(t, "existing-secret", got.SecretKey)
	require.Equal(t, "project-a", got.ProjectName)
}

func TestPublicVolcAssetSettingsRedactsSecret(t *testing.T) {
	cfg := VolcAssetSettings{
		AccessKey:   "ak-visible",
		SecretKey:   "must-not-leak",
		Region:      "ap-southeast-1",
		ProjectName: "default",
		GroupType:   "AIGC",
	}

	public := cfg.Public()

	require.Equal(t, "ak-visible", public.AccessKey)
	require.True(t, public.SecretKeyConfigured)
	encoded, err := common.Marshal(public)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(encoded), "must-not-leak"))
}
