// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	context "context"

	models "github.com/maynagashev/gophkeeper/models"
	mock "github.com/stretchr/testify/mock"
)

// VaultVersionRepository is an autogenerated mock type for the VaultVersionRepository type
type VaultVersionRepository struct {
	mock.Mock
}

type VaultVersionRepository_Expecter struct {
	mock *mock.Mock
}

func (_m *VaultVersionRepository) EXPECT() *VaultVersionRepository_Expecter {
	return &VaultVersionRepository_Expecter{mock: &_m.Mock}
}

// CreateVersion provides a mock function with given fields: ctx, version
func (_m *VaultVersionRepository) CreateVersion(ctx context.Context, version *models.VaultVersion) (int64, error) {
	ret := _m.Called(ctx, version)

	if len(ret) == 0 {
		panic("no return value specified for CreateVersion")
	}

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.VaultVersion) (int64, error)); ok {
		return rf(ctx, version)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.VaultVersion) int64); ok {
		r0 = rf(ctx, version)
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.VaultVersion) error); ok {
		r1 = rf(ctx, version)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// VaultVersionRepository_CreateVersion_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateVersion'
type VaultVersionRepository_CreateVersion_Call struct {
	*mock.Call
}

// CreateVersion is a helper method to define mock.On call
//   - ctx context.Context
//   - version *models.VaultVersion
func (_e *VaultVersionRepository_Expecter) CreateVersion(ctx interface{}, version interface{}) *VaultVersionRepository_CreateVersion_Call {
	return &VaultVersionRepository_CreateVersion_Call{Call: _e.mock.On("CreateVersion", ctx, version)}
}

func (_c *VaultVersionRepository_CreateVersion_Call) Run(run func(ctx context.Context, version *models.VaultVersion)) *VaultVersionRepository_CreateVersion_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*models.VaultVersion))
	})
	return _c
}

func (_c *VaultVersionRepository_CreateVersion_Call) Return(_a0 int64, _a1 error) *VaultVersionRepository_CreateVersion_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *VaultVersionRepository_CreateVersion_Call) RunAndReturn(run func(context.Context, *models.VaultVersion) (int64, error)) *VaultVersionRepository_CreateVersion_Call {
	_c.Call.Return(run)
	return _c
}

// GetVersionByID provides a mock function with given fields: ctx, versionID
func (_m *VaultVersionRepository) GetVersionByID(ctx context.Context, versionID int64) (*models.VaultVersion, error) {
	ret := _m.Called(ctx, versionID)

	if len(ret) == 0 {
		panic("no return value specified for GetVersionByID")
	}

	var r0 *models.VaultVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) (*models.VaultVersion, error)); ok {
		return rf(ctx, versionID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64) *models.VaultVersion); ok {
		r0 = rf(ctx, versionID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.VaultVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(ctx, versionID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// VaultVersionRepository_GetVersionByID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetVersionByID'
type VaultVersionRepository_GetVersionByID_Call struct {
	*mock.Call
}

// GetVersionByID is a helper method to define mock.On call
//   - ctx context.Context
//   - versionID int64
func (_e *VaultVersionRepository_Expecter) GetVersionByID(ctx interface{}, versionID interface{}) *VaultVersionRepository_GetVersionByID_Call {
	return &VaultVersionRepository_GetVersionByID_Call{Call: _e.mock.On("GetVersionByID", ctx, versionID)}
}

func (_c *VaultVersionRepository_GetVersionByID_Call) Run(run func(ctx context.Context, versionID int64)) *VaultVersionRepository_GetVersionByID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(int64))
	})
	return _c
}

func (_c *VaultVersionRepository_GetVersionByID_Call) Return(_a0 *models.VaultVersion, _a1 error) *VaultVersionRepository_GetVersionByID_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *VaultVersionRepository_GetVersionByID_Call) RunAndReturn(run func(context.Context, int64) (*models.VaultVersion, error)) *VaultVersionRepository_GetVersionByID_Call {
	_c.Call.Return(run)
	return _c
}

// ListVersionsByVaultID provides a mock function with given fields: ctx, vaultID, limit, offset
func (_m *VaultVersionRepository) ListVersionsByVaultID(ctx context.Context, vaultID int64, limit int, offset int) ([]models.VaultVersion, error) {
	ret := _m.Called(ctx, vaultID, limit, offset)

	if len(ret) == 0 {
		panic("no return value specified for ListVersionsByVaultID")
	}

	var r0 []models.VaultVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, int, int) ([]models.VaultVersion, error)); ok {
		return rf(ctx, vaultID, limit, offset)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64, int, int) []models.VaultVersion); ok {
		r0 = rf(ctx, vaultID, limit, offset)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.VaultVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64, int, int) error); ok {
		r1 = rf(ctx, vaultID, limit, offset)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// VaultVersionRepository_ListVersionsByVaultID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListVersionsByVaultID'
type VaultVersionRepository_ListVersionsByVaultID_Call struct {
	*mock.Call
}

// ListVersionsByVaultID is a helper method to define mock.On call
//   - ctx context.Context
//   - vaultID int64
//   - limit int
//   - offset int
func (_e *VaultVersionRepository_Expecter) ListVersionsByVaultID(ctx interface{}, vaultID interface{}, limit interface{}, offset interface{}) *VaultVersionRepository_ListVersionsByVaultID_Call {
	return &VaultVersionRepository_ListVersionsByVaultID_Call{Call: _e.mock.On("ListVersionsByVaultID", ctx, vaultID, limit, offset)}
}

func (_c *VaultVersionRepository_ListVersionsByVaultID_Call) Run(run func(ctx context.Context, vaultID int64, limit int, offset int)) *VaultVersionRepository_ListVersionsByVaultID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(int64), args[2].(int), args[3].(int))
	})
	return _c
}

func (_c *VaultVersionRepository_ListVersionsByVaultID_Call) Return(_a0 []models.VaultVersion, _a1 error) *VaultVersionRepository_ListVersionsByVaultID_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *VaultVersionRepository_ListVersionsByVaultID_Call) RunAndReturn(run func(context.Context, int64, int, int) ([]models.VaultVersion, error)) *VaultVersionRepository_ListVersionsByVaultID_Call {
	_c.Call.Return(run)
	return _c
}

// NewVaultVersionRepository creates a new instance of VaultVersionRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewVaultVersionRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *VaultVersionRepository {
	mock := &VaultVersionRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
