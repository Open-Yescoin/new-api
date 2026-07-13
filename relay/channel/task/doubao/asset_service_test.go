package doubao

import (
	"context"
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type assetAPICall struct {
	action  string
	request any
}

type fakeAssetAPI struct {
	calls     []assetAPICall
	responses map[string]any
	errors    map[string]error
}

func (f *fakeAssetAPI) Call(_ context.Context, action string, request any, result any) error {
	f.calls = append(f.calls, assetAPICall{action: action, request: request})
	if err := f.errors[action]; err != nil {
		return err
	}
	response, ok := f.responses[action]
	if !ok || result == nil {
		return nil
	}
	data, err := common.Marshal(response)
	if err != nil {
		return err
	}
	return common.Unmarshal(data, result)
}

type fakeAssetGroupRepository struct {
	groups map[int]string
}

func (f *fakeAssetGroupRepository) Get(userId int) (string, error) {
	groupId, ok := f.groups[userId]
	if !ok {
		return "", gorm.ErrRecordNotFound
	}
	return groupId, nil
}

func (f *fakeAssetGroupRepository) Save(userId int, groupId string) error {
	f.groups[userId] = groupId
	return nil
}

func newAssetServiceTestFixture() (*AssetService, *fakeAssetAPI, *fakeAssetGroupRepository) {
	api := &fakeAssetAPI{responses: map[string]any{}, errors: map[string]error{}}
	groups := &fakeAssetGroupRepository{groups: map[int]string{}}
	cfg := system_setting.VolcAssetSettings{ProjectName: "project-a", GroupType: "LivenessFace"}
	return NewAssetService(api, groups, cfg), api, groups
}

func TestAssetServiceCreatesVisualValidationSession(t *testing.T) {
	service, api, _ := newAssetServiceTestFixture()
	api.responses["CreateVisualValidateSession"] = CreateVisualValidateSessionResponse{
		BytedToken: "token-1",
		H5Link:     "https://verify.example/session",
	}

	got, err := service.CreateVisualValidateSession(context.Background(), 42, CreateVisualValidateSessionRequest{
		CallbackURL: "https://app.example/seedance/callback",
	})

	require.NoError(t, err)
	require.Equal(t, "token-1", got.BytedToken)
	require.Len(t, api.calls, 1)
	require.Equal(t, "CreateVisualValidateSession", api.calls[0].action)
	request := api.calls[0].request.(CreateVisualValidateSessionRequest)
	require.Equal(t, "project-a", request.ProjectName)
}

func TestAssetServiceStoresVerifiedRealPersonGroup(t *testing.T) {
	service, api, groups := newAssetServiceTestFixture()
	api.responses["GetVisualValidateResult"] = GetVisualValidateResultResponse{GroupId: "group-real-42"}

	got, err := service.GetVisualValidateResult(context.Background(), 42, GetVisualValidateResultRequest{
		BytedToken: "token-1",
	})

	require.NoError(t, err)
	require.Equal(t, "group-real-42", got.GroupId)
	require.Equal(t, "group-real-42", groups.groups[42])
	request := api.calls[0].request.(GetVisualValidateResultRequest)
	require.Equal(t, "project-a", request.ProjectName)
}

func TestAssetServiceRequiresRealPersonAuthorizationBeforeCreate(t *testing.T) {
	service, api, _ := newAssetServiceTestFixture()

	_, err := service.CreateAsset(context.Background(), 42, CreateAssetRequest{
		URL:       "https://cdn.example/person.jpg",
		AssetType: "Image",
	})

	require.ErrorIs(t, err, ErrAssetGroupNotAuthorized)
	require.Empty(t, api.calls)
}

func TestAssetServiceCreatesImageAndVideoInVerifiedGroup(t *testing.T) {
	for _, assetType := range []string{"Image", "Video"} {
		t.Run(assetType, func(t *testing.T) {
			service, api, groups := newAssetServiceTestFixture()
			groups.groups[42] = "group-real-42"
			api.responses["CreateAsset"] = CreateAssetResponse{Id: "asset-1", Status: "Processing"}

			got, err := service.CreateAsset(context.Background(), 42, CreateAssetRequest{
				URL:       "https://cdn.example/person-media",
				AssetType: assetType,
			})

			require.NoError(t, err)
			require.Equal(t, "asset-1", got.Id)
			request := api.calls[0].request.(CreateAssetRequest)
			require.Equal(t, "group-real-42", request.GroupId)
			require.Equal(t, "project-a", request.ProjectName)
			require.Equal(t, assetType, request.AssetType)
		})
	}
}

func TestAssetServiceRejectsInvalidAssetInput(t *testing.T) {
	tests := []CreateAssetRequest{
		{URL: "file:///tmp/person.jpg", AssetType: "Image"},
		{URL: "https://cdn.example/person.jpg", AssetType: "Archive"},
	}
	for _, request := range tests {
		service, api, groups := newAssetServiceTestFixture()
		groups.groups[42] = "group-real-42"

		_, err := service.CreateAsset(context.Background(), 42, request)

		require.ErrorIs(t, err, ErrInvalidAssetRequest)
		require.Empty(t, api.calls)
	}
}

func TestAssetServiceForcesListToVerifiedGroup(t *testing.T) {
	service, api, groups := newAssetServiceTestFixture()
	groups.groups[42] = "group-real-42"
	api.responses["ListAssets"] = ListAssetsResponse{}

	_, err := service.ListAssets(context.Background(), 42, ListAssetsRequest{
		Filter: &AssetFilter{GroupIds: []string{"group-other"}},
	})

	require.NoError(t, err)
	request := api.calls[0].request.(ListAssetsRequest)
	require.Equal(t, []string{"group-real-42"}, request.Filter.GroupIds)
	require.Equal(t, "LivenessFace", request.Filter.GroupType)
	require.Equal(t, "project-a", request.ProjectName)
}

func TestAssetServiceRejectsCrossGroupMutations(t *testing.T) {
	operations := []struct {
		name string
		run  func(*AssetService) error
	}{
		{name: "get", run: func(service *AssetService) error {
			_, err := service.GetAsset(context.Background(), 42, GetAssetRequest{Id: "asset-other"})
			return err
		}},
		{name: "update", run: func(service *AssetService) error {
			return service.UpdateAsset(context.Background(), 42, UpdateAssetRequest{Id: "asset-other", Name: "renamed"})
		}},
		{name: "delete", run: func(service *AssetService) error {
			return service.DeleteAsset(context.Background(), 42, DeleteAssetRequest{Id: "asset-other"})
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			service, api, groups := newAssetServiceTestFixture()
			groups.groups[42] = "group-real-42"
			api.responses["GetAsset"] = AssetItem{Id: "asset-other", GroupId: "group-other"}

			err := operation.run(service)

			require.True(t, errors.Is(err, ErrAssetNotFound))
			require.Len(t, api.calls, 1)
			require.Equal(t, "GetAsset", api.calls[0].action)
		})
	}
}
