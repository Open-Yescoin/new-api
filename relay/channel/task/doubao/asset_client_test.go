package doubao

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func fixedAssetClientTime() time.Time {
	return time.Date(2026, 7, 12, 1, 2, 3, 0, time.UTC)
}

type assetRoundTripFunc func(*http.Request) (*http.Response, error)

func (f assetRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestVolcAssetClientSignsAndDecodesResult(t *testing.T) {
	var captured *http.Request
	httpClient := &http.Client{Transport: assetRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		captured = r.Clone(r.Context())
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
            "ResponseMetadata":{"RequestId":"request-1"},
            "Result":{"Id":"asset-1"}
        }`)),
		}, nil
	})}

	cfg := system_setting.VolcAssetSettings{
		AccessKey: "ak-test",
		SecretKey: "sk-test",
		Region:    "ap-southeast-1",
	}
	client := newVolcAssetClient(cfg, "https://ark.test", httpClient, fixedAssetClientTime)
	var result struct {
		Id string `json:"Id"`
	}

	err := client.Call(context.Background(), "CreateAsset", map[string]any{"URL": "https://cdn.example/a.jpg"}, &result)

	require.NoError(t, err)
	require.Equal(t, "asset-1", result.Id)
	require.NotNil(t, captured)
	require.Equal(t, "CreateAsset", captured.URL.Query().Get("Action"))
	require.Equal(t, "2024-01-01", captured.URL.Query().Get("Version"))
	require.Equal(t, "20260712T010203Z", captured.Header.Get("X-Date"))
	require.NotEmpty(t, captured.Header.Get("X-Content-Sha256"))
	require.Contains(t, captured.Header.Get("Authorization"), "Credential=ak-test/20260712/ap-southeast-1/ark/request")
	require.Contains(t, captured.Header.Get("Authorization"), "SignedHeaders=content-type;host;x-content-sha256;x-date")
}

func TestVolcAssetClientReturnsUpstreamBusinessError(t *testing.T) {
	httpClient := &http.Client{Transport: assetRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
            "ResponseMetadata":{"Error":{"Code":"SubscriptionRequired","Message":"subscribe first"}}
        }`)),
		}, nil
	})}

	cfg := system_setting.VolcAssetSettings{AccessKey: "ak", SecretKey: "sk", Region: "ap-southeast-1"}
	client := newVolcAssetClient(cfg, "https://ark.test", httpClient, fixedAssetClientTime)

	err := client.Call(context.Background(), "CreateAssetGroup", map[string]any{"Name": "group"}, &struct{}{})

	var apiErr *AssetAPIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	require.Equal(t, "SubscriptionRequired", apiErr.Code)
	require.Equal(t, "subscribe first", apiErr.Message)
}

func TestVolcAssetClientRejectsMissingCredentials(t *testing.T) {
	client := newVolcAssetClient(system_setting.VolcAssetSettings{}, "https://example.invalid", http.DefaultClient, fixedAssetClientTime)

	err := client.Call(context.Background(), "ListAssets", map[string]any{}, &struct{}{})

	require.ErrorIs(t, err, ErrVolcAssetNotConfigured)
	require.False(t, strings.Contains(err.Error(), "secret"))
}
