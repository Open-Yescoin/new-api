package doubao

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const (
	assetServiceName     = "ark"
	assetAPIVersion      = "2024-01-01"
	maxAssetResponseSize = 10 << 20
)

var ErrVolcAssetNotConfigured = errors.New("BytePlus asset library is not configured")

type AssetAPI interface {
	Call(ctx context.Context, action string, request any, result any) error
}

type AssetAPIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *AssetAPIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type VolcAssetClient struct {
	config     system_setting.VolcAssetSettings
	baseURL    string
	httpClient *http.Client
	now        func() time.Time
}

func NewVolcAssetClient(config system_setting.VolcAssetSettings, httpClient *http.Client) *VolcAssetClient {
	return newVolcAssetClient(config, config.GetBaseURL(), httpClient, time.Now)
}

func newVolcAssetClient(config system_setting.VolcAssetSettings, baseURL string, httpClient *http.Client, now func() time.Time) *VolcAssetClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if now == nil {
		now = time.Now
	}
	return &VolcAssetClient{
		config:     config,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		now:        now,
	}
}

func (c *VolcAssetClient) Call(ctx context.Context, action string, request any, result any) error {
	if c.config.AccessKey == "" || c.config.SecretKey == "" {
		return ErrVolcAssetNotConfigured
	}
	body, err := common.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal asset request: %w", err)
	}
	endpoint := fmt.Sprintf("%s/?Action=%s&Version=%s", c.baseURL, url.QueryEscape(action), assetAPIVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create asset request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	c.sign(req, body)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call BytePlus asset API: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxAssetResponseSize+1))
	if err != nil {
		return fmt.Errorf("read BytePlus asset response: %w", err)
	}
	if len(responseBody) > maxAssetResponseSize {
		return fmt.Errorf("BytePlus asset response exceeds %d bytes", maxAssetResponseSize)
	}

	var envelope struct {
		ResponseMetadata struct {
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"ResponseMetadata"`
		Result json.RawMessage `json:"Result"`
	}
	if err := common.Unmarshal(responseBody, &envelope); err != nil {
		return &AssetAPIError{
			StatusCode: resp.StatusCode,
			Code:       "invalid_upstream_response",
			Message:    "BytePlus asset API returned invalid JSON",
		}
	}
	if envelope.ResponseMetadata.Error.Code != "" {
		return &AssetAPIError{
			StatusCode: resp.StatusCode,
			Code:       envelope.ResponseMetadata.Error.Code,
			Message:    envelope.ResponseMetadata.Error.Message,
		}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &AssetAPIError{
			StatusCode: resp.StatusCode,
			Code:       "upstream_http_error",
			Message:    http.StatusText(resp.StatusCode),
		}
	}
	if result == nil || len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return nil
	}
	if err := common.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode BytePlus asset result: %w", err)
	}
	return nil
}

func (c *VolcAssetClient) sign(req *http.Request, body []byte) {
	payloadHash := hex.EncodeToString(common.Sha256Raw(body))
	now := c.now().UTC()
	xDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")

	req.Header.Set("X-Date", xDate)
	req.Header.Set("X-Content-Sha256", payloadHash)

	query := req.URL.Query()
	queryKeys := make([]string, 0, len(query))
	for key := range query {
		queryKeys = append(queryKeys, key)
	}
	sort.Strings(queryKeys)
	queryParts := make([]string, 0, len(queryKeys))
	for _, key := range queryKeys {
		values := query[key]
		sort.Strings(values)
		for _, value := range values {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(value)))
		}
	}

	headers := map[string]string{
		"content-type":     req.Header.Get("Content-Type"),
		"host":             req.URL.Host,
		"x-content-sha256": payloadHash,
		"x-date":           xDate,
	}
	headerKeys := make([]string, 0, len(headers))
	for key := range headers {
		headerKeys = append(headerKeys, key)
	}
	sort.Strings(headerKeys)
	var canonicalHeaders strings.Builder
	for _, key := range headerKeys {
		canonicalHeaders.WriteString(key)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(headers[key]))
		canonicalHeaders.WriteString("\n")
	}
	signedHeaders := strings.Join(headerKeys, ";")
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.EscapedPath(),
		strings.Join(queryParts, "&"),
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	)
	canonicalRequestHash := hex.EncodeToString(common.Sha256Raw([]byte(canonicalRequest)))
	credentialScope := fmt.Sprintf("%s/%s/%s/request", shortDate, c.config.GetRegion(), assetServiceName)
	stringToSign := fmt.Sprintf("HMAC-SHA256\n%s\n%s\n%s", xDate, credentialScope, canonicalRequestHash)

	kDate := common.HmacSha256Raw([]byte(shortDate), []byte(c.config.SecretKey))
	kRegion := common.HmacSha256Raw([]byte(c.config.GetRegion()), kDate)
	kService := common.HmacSha256Raw([]byte(assetServiceName), kRegion)
	kSigning := common.HmacSha256Raw([]byte("request"), kService)
	signature := hex.EncodeToString(common.HmacSha256Raw([]byte(stringToSign), kSigning))

	req.Header.Set("Authorization", fmt.Sprintf(
		"HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.config.AccessKey,
		credentialScope,
		signedHeaders,
		signature,
	))
}
