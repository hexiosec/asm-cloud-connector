package aws

import (
	"context"
	"testing"

	cloud_provider_t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAWSProvider_GetAPIKey_NoSecret(t *testing.T) {
	provider := &AWSProvider{
		cfg: &config.AWSCloudProvider{},
	}

	_, err := provider.GetAPIKey(context.Background())

	assert.ErrorIs(t, err, cloud_provider_t.ErrNoAPIKey)
}

func TestAWSProvider_GetAPIKey_ReturnsSecret(t *testing.T) {
	secretName := "my-secret"
	provider, mockWrapper := newProviderWithMock(t, &config.AWSCloudProvider{
		APIKeySecret: &secretName,
	})

	mockWrapper.On("GetSecretString", secretName).Return("secret-value", nil)

	value, err := provider.GetAPIKey(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "secret-value", value)
}

func TestAWSProvider_GetResources_DefaultAccountConfig(t *testing.T) {
	cfg := &config.AWSCloudProvider{
		Services: &config.AWSServices{CheckEC2: true},
	}
	provider, mockWrapper := newProviderWithMock(t, cfg)

	mockWrapper.On("GetRegions").Return([]string{"us-east-1"}, nil)
	mockWrapper.On("ChangeRegion", "us-east-1").Return()
	mockWrapper.On("GetEC2Resources", mock.Anything, true).Return(assetSlice("i-1"), nil).Once()
	mockWrapper.On("ResetRegion").Return()

	resources, err := provider.GetResources(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, assetSlice("i-1"), resources)
}

func TestAWSProvider_GetResources_ListAllAccountsError(t *testing.T) {
	role := "my-role"
	cfg := &config.AWSCloudProvider{
		ListAllAccounts: true,
		AssumeRole:      &role,
		Services:        &config.AWSServices{CheckEC2: true},
	}
	provider, mockWrapper := newProviderWithMock(t, cfg)

	mockWrapper.On("ListAllAccounts").Return(nil, assert.AnError)

	_, err := provider.GetResources(context.Background())
	assert.ErrorIs(t, err, assert.AnError)
}

func TestAWSProvider_GetResources_AssumeRolePerAccount(t *testing.T) {
	role := "MyRole"
	account := "123456789012"
	cfg := &config.AWSCloudProvider{
		Accounts:   []string{account},
		AssumeRole: &role,
		Services:   &config.AWSServices{CheckEC2: true},
	}

	parent := NewMockWrapper(t).(*MockWrapper)
	provider := &AWSProvider{
		cfg:     cfg,
		wrapper: parent,
	}

	child := NewMockWrapper(t).(*MockWrapper)

	parent.On("AssumeRole", "arn:aws:iam::123456789012:role/MyRole").
		Return(child, nil)

	child.On("GetRegions").Return([]string{"us-east-1"}, nil)
	child.On("ChangeRegion", "us-east-1").Return()
	child.On("GetEC2Resources", mock.Anything, true).Return(assetSlice("acct-res"), nil).Once()
	child.On("ResetRegion").Return()

	resources, err := provider.GetResources(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, assetSlice("acct-res"), resources)
}

func TestAWSProvider_GetResources_AssumeRoleErr_Continue(t *testing.T) {
	role := "MyRole"
	account := "123456789012"
	cfg := &config.AWSCloudProvider{
		Accounts:   []string{account},
		AssumeRole: &role,
		Services:   &config.AWSServices{CheckEC2: true},
	}

	parent := NewMockWrapper(t).(*MockWrapper)
	provider := &AWSProvider{
		cfg:     cfg,
		wrapper: parent,
	}

	parent.On("AssumeRole", "arn:aws:iam::123456789012:role/MyRole").
		Return(nil, assert.AnError)

	resources, err := provider.GetResources(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, resources)
}

func Test_getResources_GetRegionsError(t *testing.T) {
	mockWrapper := NewMockWrapper(t).(*MockWrapper)
	services := &config.AWSServices{CheckEC2: true}

	mockWrapper.On("GetRegions").Return(nil, assert.AnError)

	_, err := getResources(context.Background(), mockWrapper, services, nil, true)
	assert.ErrorContains(t, err, "could not determine active regions")
	assert.ErrorIs(t, err, assert.AnError)
}

func Test_getResources_AggregatesResourcesAcrossRegions(t *testing.T) {
	mockWrapper := NewMockWrapper(t).(*MockWrapper)
	services := &config.AWSServices{CheckEC2: true}

	mockWrapper.On("GetRegions").Return([]string{"us-east-1", "us-west-2"}, nil)
	mockWrapper.On("ChangeRegion", "us-east-1").Return()
	mockWrapper.On("ChangeRegion", "us-west-2").Return()
	mockWrapper.On("GetEC2Resources", mock.Anything, true).Return(assetSlice("res-east"), nil).Once()
	mockWrapper.On("GetEC2Resources", mock.Anything, true).Return(assetSlice("res-east", "res-west"), nil).Once()
	mockWrapper.On("ResetRegion").Return()

	resources, err := getResources(context.Background(), mockWrapper, services, []*cloud_provider_t.Asset{}, true)
	assert.NoError(t, err)
	assert.Equal(t, assetSlice("res-east", "res-west"), resources)
}

func TestAWSProvider_GetResources_PopulateTagsDisabled_PassesFalse(t *testing.T) {
	disabled := false
	cfg := &config.AWSCloudProvider{
		PopulateTags: &disabled,
		Services:     &config.AWSServices{CheckEC2: true},
	}
	provider, mockWrapper := newProviderWithMock(t, cfg)

	mockWrapper.On("GetRegions").Return([]string{"us-east-1"}, nil)
	mockWrapper.On("ChangeRegion", "us-east-1").Return()
	mockWrapper.On("GetEC2Resources", mock.Anything, false).Return(assetSlice("i-1"), nil).Once()
	mockWrapper.On("ResetRegion").Return()

	resources, err := provider.GetResources(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, assetSlice("i-1"), resources)
}

func newProviderWithMock(t *testing.T, cfg *config.AWSCloudProvider) (*AWSProvider, *MockWrapper) {
	t.Helper()
	wrapper := NewMockWrapper(t).(*MockWrapper)
	return &AWSProvider{
		cfg:     cfg,
		wrapper: wrapper,
	}, wrapper
}

func assetSlice(names ...string) []*cloud_provider_t.Asset {
	assets := make([]*cloud_provider_t.Asset, 0, len(names))
	for _, name := range names {
		assets = append(assets, &cloud_provider_t.Asset{Name: name})
	}
	return assets
}
