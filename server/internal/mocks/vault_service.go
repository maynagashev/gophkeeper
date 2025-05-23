// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	io "io"

	models "github.com/maynagashev/gophkeeper/models"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// VaultService is an autogenerated mock type for the VaultService type
type VaultService struct {
	mock.Mock
}

type VaultService_Expecter struct {
	mock *mock.Mock
}

func (_m *VaultService) EXPECT() *VaultService_Expecter {
	return &VaultService_Expecter{mock: &_m.Mock}
}

// DownloadVault provides a mock function with given fields: userID
func (_m *VaultService) DownloadVault(userID int64) (io.ReadCloser, *models.VaultVersion, error) {
	ret := _m.Called(userID)

	if len(ret) == 0 {
		panic("no return value specified for DownloadVault")
	}

	var r0 io.ReadCloser
	var r1 *models.VaultVersion
	var r2 error
	if rf, ok := ret.Get(0).(func(int64) (io.ReadCloser, *models.VaultVersion, error)); ok {
		return rf(userID)
	}
	if rf, ok := ret.Get(0).(func(int64) io.ReadCloser); ok {
		r0 = rf(userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.ReadCloser)
		}
	}

	if rf, ok := ret.Get(1).(func(int64) *models.VaultVersion); ok {
		r1 = rf(userID)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*models.VaultVersion)
		}
	}

	if rf, ok := ret.Get(2).(func(int64) error); ok {
		r2 = rf(userID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// VaultService_DownloadVault_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DownloadVault'
type VaultService_DownloadVault_Call struct {
	*mock.Call
}

// DownloadVault is a helper method to define mock.On call
//   - userID int64
func (_e *VaultService_Expecter) DownloadVault(userID interface{}) *VaultService_DownloadVault_Call {
	return &VaultService_DownloadVault_Call{Call: _e.mock.On("DownloadVault", userID)}
}

func (_c *VaultService_DownloadVault_Call) Run(run func(userID int64)) *VaultService_DownloadVault_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int64))
	})
	return _c
}

func (_c *VaultService_DownloadVault_Call) Return(_a0 io.ReadCloser, _a1 *models.VaultVersion, _a2 error) *VaultService_DownloadVault_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *VaultService_DownloadVault_Call) RunAndReturn(run func(int64) (io.ReadCloser, *models.VaultVersion, error)) *VaultService_DownloadVault_Call {
	_c.Call.Return(run)
	return _c
}

// GetVaultMetadata provides a mock function with given fields: userID
func (_m *VaultService) GetVaultMetadata(userID int64) (*models.VaultVersion, error) {
	ret := _m.Called(userID)

	if len(ret) == 0 {
		panic("no return value specified for GetVaultMetadata")
	}

	var r0 *models.VaultVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(int64) (*models.VaultVersion, error)); ok {
		return rf(userID)
	}
	if rf, ok := ret.Get(0).(func(int64) *models.VaultVersion); ok {
		r0 = rf(userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.VaultVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(int64) error); ok {
		r1 = rf(userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// VaultService_GetVaultMetadata_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetVaultMetadata'
type VaultService_GetVaultMetadata_Call struct {
	*mock.Call
}

// GetVaultMetadata is a helper method to define mock.On call
//   - userID int64
func (_e *VaultService_Expecter) GetVaultMetadata(userID interface{}) *VaultService_GetVaultMetadata_Call {
	return &VaultService_GetVaultMetadata_Call{Call: _e.mock.On("GetVaultMetadata", userID)}
}

func (_c *VaultService_GetVaultMetadata_Call) Run(run func(userID int64)) *VaultService_GetVaultMetadata_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int64))
	})
	return _c
}

func (_c *VaultService_GetVaultMetadata_Call) Return(_a0 *models.VaultVersion, _a1 error) *VaultService_GetVaultMetadata_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *VaultService_GetVaultMetadata_Call) RunAndReturn(run func(int64) (*models.VaultVersion, error)) *VaultService_GetVaultMetadata_Call {
	_c.Call.Return(run)
	return _c
}

// ListVersions provides a mock function with given fields: userID, limit, offset
func (_m *VaultService) ListVersions(userID int64, limit int, offset int) ([]models.VaultVersion, error) {
	ret := _m.Called(userID, limit, offset)

	if len(ret) == 0 {
		panic("no return value specified for ListVersions")
	}

	var r0 []models.VaultVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(int64, int, int) ([]models.VaultVersion, error)); ok {
		return rf(userID, limit, offset)
	}
	if rf, ok := ret.Get(0).(func(int64, int, int) []models.VaultVersion); ok {
		r0 = rf(userID, limit, offset)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.VaultVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(int64, int, int) error); ok {
		r1 = rf(userID, limit, offset)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// VaultService_ListVersions_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListVersions'
type VaultService_ListVersions_Call struct {
	*mock.Call
}

// ListVersions is a helper method to define mock.On call
//   - userID int64
//   - limit int
//   - offset int
func (_e *VaultService_Expecter) ListVersions(userID interface{}, limit interface{}, offset interface{}) *VaultService_ListVersions_Call {
	return &VaultService_ListVersions_Call{Call: _e.mock.On("ListVersions", userID, limit, offset)}
}

func (_c *VaultService_ListVersions_Call) Run(run func(userID int64, limit int, offset int)) *VaultService_ListVersions_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int64), args[1].(int), args[2].(int))
	})
	return _c
}

func (_c *VaultService_ListVersions_Call) Return(_a0 []models.VaultVersion, _a1 error) *VaultService_ListVersions_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *VaultService_ListVersions_Call) RunAndReturn(run func(int64, int, int) ([]models.VaultVersion, error)) *VaultService_ListVersions_Call {
	_c.Call.Return(run)
	return _c
}

// RollbackToVersion provides a mock function with given fields: userID, versionID
func (_m *VaultService) RollbackToVersion(userID int64, versionID int64) error {
	ret := _m.Called(userID, versionID)

	if len(ret) == 0 {
		panic("no return value specified for RollbackToVersion")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(int64, int64) error); ok {
		r0 = rf(userID, versionID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// VaultService_RollbackToVersion_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RollbackToVersion'
type VaultService_RollbackToVersion_Call struct {
	*mock.Call
}

// RollbackToVersion is a helper method to define mock.On call
//   - userID int64
//   - versionID int64
func (_e *VaultService_Expecter) RollbackToVersion(userID interface{}, versionID interface{}) *VaultService_RollbackToVersion_Call {
	return &VaultService_RollbackToVersion_Call{Call: _e.mock.On("RollbackToVersion", userID, versionID)}
}

func (_c *VaultService_RollbackToVersion_Call) Run(run func(userID int64, versionID int64)) *VaultService_RollbackToVersion_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int64), args[1].(int64))
	})
	return _c
}

func (_c *VaultService_RollbackToVersion_Call) Return(_a0 error) *VaultService_RollbackToVersion_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *VaultService_RollbackToVersion_Call) RunAndReturn(run func(int64, int64) error) *VaultService_RollbackToVersion_Call {
	_c.Call.Return(run)
	return _c
}

// UploadVault provides a mock function with given fields: userID, reader, size, contentType, contentModifiedAt
func (_m *VaultService) UploadVault(userID int64, reader io.Reader, size int64, contentType string, contentModifiedAt time.Time) error {
	ret := _m.Called(userID, reader, size, contentType, contentModifiedAt)

	if len(ret) == 0 {
		panic("no return value specified for UploadVault")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(int64, io.Reader, int64, string, time.Time) error); ok {
		r0 = rf(userID, reader, size, contentType, contentModifiedAt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// VaultService_UploadVault_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UploadVault'
type VaultService_UploadVault_Call struct {
	*mock.Call
}

// UploadVault is a helper method to define mock.On call
//   - userID int64
//   - reader io.Reader
//   - size int64
//   - contentType string
//   - contentModifiedAt time.Time
func (_e *VaultService_Expecter) UploadVault(userID interface{}, reader interface{}, size interface{}, contentType interface{}, contentModifiedAt interface{}) *VaultService_UploadVault_Call {
	return &VaultService_UploadVault_Call{Call: _e.mock.On("UploadVault", userID, reader, size, contentType, contentModifiedAt)}
}

func (_c *VaultService_UploadVault_Call) Run(run func(userID int64, reader io.Reader, size int64, contentType string, contentModifiedAt time.Time)) *VaultService_UploadVault_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int64), args[1].(io.Reader), args[2].(int64), args[3].(string), args[4].(time.Time))
	})
	return _c
}

func (_c *VaultService_UploadVault_Call) Return(_a0 error) *VaultService_UploadVault_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *VaultService_UploadVault_Call) RunAndReturn(run func(int64, io.Reader, int64, string, time.Time) error) *VaultService_UploadVault_Call {
	_c.Call.Return(run)
	return _c
}

// NewVaultService creates a new instance of VaultService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewVaultService(t interface {
	mock.TestingT
	Cleanup(func())
}) *VaultService {
	mock := &VaultService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
