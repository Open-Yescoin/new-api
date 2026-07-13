package system_setting

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var VolcAssetConfig = VolcAssetSettings{}

type VolcAssetSettings struct {
	AccessKey                    string `json:"access_key"`
	SecretKey                    string `json:"secret_key,omitempty"`
	Region                       string `json:"region"`
	ProjectName                  string `json:"project_name"`
	GroupType                    string `json:"group_type"`
	AuthorizationCallbackBaseURL string `json:"authorization_callback_base_url"`
}

type PublicVolcAssetSettings struct {
	AccessKey                    string `json:"access_key"`
	SecretKeyConfigured          bool   `json:"secret_key_configured"`
	Region                       string `json:"region"`
	ProjectName                  string `json:"project_name"`
	GroupType                    string `json:"group_type"`
	AuthorizationCallbackBaseURL string `json:"authorization_callback_base_url"`
}

func (v VolcAssetSettings) GetRegion() string {
	if v.Region == "" {
		return "ap-southeast-1"
	}
	return v.Region
}

func (v VolcAssetSettings) GetProjectName() string {
	if v.ProjectName == "" {
		return "default"
	}
	return v.ProjectName
}

func (v VolcAssetSettings) GetGroupType() string {
	if v.GroupType == "" {
		return "LivenessFace"
	}
	return v.GroupType
}

func (v VolcAssetSettings) GetBaseURL() string {
	return fmt.Sprintf("https://ark.%s.byteplusapi.com", v.GetRegion())
}

func (v VolcAssetSettings) GetAuthorizationCallbackURL() (string, error) {
	raw := strings.TrimSpace(v.AuthorizationCallbackBaseURL)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" ||
		parsed.Fragment != "" || parsed.User != nil {
		return "", errors.New("a valid HTTPS Seedance authorization callback origin is required")
	}
	return strings.TrimRight(raw, "/") + "/seedance/authorization/callback", nil
}

func (v VolcAssetSettings) Public() PublicVolcAssetSettings {
	return PublicVolcAssetSettings{
		AccessKey:                    v.AccessKey,
		SecretKeyConfigured:          v.SecretKey != "",
		Region:                       v.GetRegion(),
		ProjectName:                  v.GetProjectName(),
		GroupType:                    v.GetGroupType(),
		AuthorizationCallbackBaseURL: strings.TrimSpace(v.AuthorizationCallbackBaseURL),
	}
}

func MergeVolcAssetSettings(current, incoming VolcAssetSettings) VolcAssetSettings {
	if incoming.SecretKey == "" {
		incoming.SecretKey = current.SecretKey
	}
	return incoming
}
