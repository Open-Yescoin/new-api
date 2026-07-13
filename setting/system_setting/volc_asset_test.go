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

func TestVolcAssetSettingsAuthorizationCallbackURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
		wantErr bool
	}{
		{
			name:    "production origin",
			baseURL: "https://www.token123.co",
			want:    "https://www.token123.co/seedance/authorization/callback",
		},
		{
			name:    "trailing slash",
			baseURL: "https://www.token123.co/",
			want:    "https://www.token123.co/seedance/authorization/callback",
		},
		{name: "http is rejected", baseURL: "http://www.token123.co", wantErr: true},
		{name: "path is rejected", baseURL: "https://www.token123.co/app", wantErr: true},
		{name: "query is rejected", baseURL: "https://www.token123.co?next=bad", wantErr: true},
		{name: "credentials are rejected", baseURL: "https://user@example.com", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := VolcAssetSettings{AuthorizationCallbackBaseURL: test.baseURL}

			got, err := cfg.GetAuthorizationCallbackURL()

			require.Equal(t, test.wantErr, err != nil)
			require.Equal(t, test.want, got)
		})
	}
}
