// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	"context"
	"io"
	"os"

	"github.com/messikiller/afero-oss/internal/utils"
	mock "github.com/stretchr/testify/mock"
)

// NewMockObjectManager creates a new instance of MockObjectManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockObjectManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockObjectManager {
	mock := &MockObjectManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// MockObjectManager is an autogenerated mock type for the ObjectManager type
type MockObjectManager struct {
	mock.Mock
}

type MockObjectManager_Expecter struct {
	mock *mock.Mock
}

func (_m *MockObjectManager) EXPECT() *MockObjectManager_Expecter {
	return &MockObjectManager_Expecter{mock: &_m.Mock}
}

// CopyObject provides a mock function for the type MockObjectManager
func (_mock *MockObjectManager) CopyObject(ctx context.Context, bucket string, srcName string, targetName string) error {
	ret := _mock.Called(ctx, bucket, srcName, targetName)

	if len(ret) == 0 {
		panic("no return value specified for CopyObject")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, string) error); ok {
		r0 = returnFunc(ctx, bucket, srcName, targetName)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockObjectManager_CopyObject_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CopyObject'
type MockObjectManager_CopyObject_Call struct {
	*mock.Call
}

// CopyObject is a helper method to define mock.On call
//   - ctx
//   - bucket
//   - srcName
//   - targetName
func (_e *MockObjectManager_Expecter) CopyObject(ctx interface{}, bucket interface{}, srcName interface{}, targetName interface{}) *MockObjectManager_CopyObject_Call {
	return &MockObjectManager_CopyObject_Call{Call: _e.mock.On("CopyObject", ctx, bucket, srcName, targetName)}
}

func (_c *MockObjectManager_CopyObject_Call) Run(run func(ctx context.Context, bucket string, srcName string, targetName string)) *MockObjectManager_CopyObject_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(string))
	})
	return _c
}

func (_c *MockObjectManager_CopyObject_Call) Return(err error) *MockObjectManager_CopyObject_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockObjectManager_CopyObject_Call) RunAndReturn(run func(ctx context.Context, bucket string, srcName string, targetName string) error) *MockObjectManager_CopyObject_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteObject provides a mock function for the type MockObjectManager
func (_mock *MockObjectManager) DeleteObject(ctx context.Context, bucket string, name string) error {
	ret := _mock.Called(ctx, bucket, name)

	if len(ret) == 0 {
		panic("no return value specified for DeleteObject")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = returnFunc(ctx, bucket, name)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockObjectManager_DeleteObject_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteObject'
type MockObjectManager_DeleteObject_Call struct {
	*mock.Call
}

// DeleteObject is a helper method to define mock.On call
//   - ctx
//   - bucket
//   - name
func (_e *MockObjectManager_Expecter) DeleteObject(ctx interface{}, bucket interface{}, name interface{}) *MockObjectManager_DeleteObject_Call {
	return &MockObjectManager_DeleteObject_Call{Call: _e.mock.On("DeleteObject", ctx, bucket, name)}
}

func (_c *MockObjectManager_DeleteObject_Call) Run(run func(ctx context.Context, bucket string, name string)) *MockObjectManager_DeleteObject_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockObjectManager_DeleteObject_Call) Return(err error) *MockObjectManager_DeleteObject_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockObjectManager_DeleteObject_Call) RunAndReturn(run func(ctx context.Context, bucket string, name string) error) *MockObjectManager_DeleteObject_Call {
	_c.Call.Return(run)
	return _c
}

// GetObject provides a mock function for the type MockObjectManager
func (_mock *MockObjectManager) GetObject(ctx context.Context, bucket string, name string) (io.Reader, utils.CleanUp, error) {
	ret := _mock.Called(ctx, bucket, name)

	if len(ret) == 0 {
		panic("no return value specified for GetObject")
	}

	var r0 io.Reader
	var r1 utils.CleanUp
	var r2 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) (io.Reader, utils.CleanUp, error)); ok {
		return returnFunc(ctx, bucket, name)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) io.Reader); ok {
		r0 = returnFunc(ctx, bucket, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.Reader)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, string) utils.CleanUp); ok {
		r1 = returnFunc(ctx, bucket, name)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(utils.CleanUp)
		}
	}
	if returnFunc, ok := ret.Get(2).(func(context.Context, string, string) error); ok {
		r2 = returnFunc(ctx, bucket, name)
	} else {
		r2 = ret.Error(2)
	}
	return r0, r1, r2
}

// MockObjectManager_GetObject_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetObject'
type MockObjectManager_GetObject_Call struct {
	*mock.Call
}

// GetObject is a helper method to define mock.On call
//   - ctx
//   - bucket
//   - name
func (_e *MockObjectManager_Expecter) GetObject(ctx interface{}, bucket interface{}, name interface{}) *MockObjectManager_GetObject_Call {
	return &MockObjectManager_GetObject_Call{Call: _e.mock.On("GetObject", ctx, bucket, name)}
}

func (_c *MockObjectManager_GetObject_Call) Run(run func(ctx context.Context, bucket string, name string)) *MockObjectManager_GetObject_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockObjectManager_GetObject_Call) Return(reader io.Reader, cleanUp utils.CleanUp, err error) *MockObjectManager_GetObject_Call {
	_c.Call.Return(reader, cleanUp, err)
	return _c
}

func (_c *MockObjectManager_GetObject_Call) RunAndReturn(run func(ctx context.Context, bucket string, name string) (io.Reader, utils.CleanUp, error)) *MockObjectManager_GetObject_Call {
	_c.Call.Return(run)
	return _c
}

// GetObjectMeta provides a mock function for the type MockObjectManager
func (_mock *MockObjectManager) GetObjectMeta(ctx context.Context, bucket string, name string) (os.FileInfo, error) {
	ret := _mock.Called(ctx, bucket, name)

	if len(ret) == 0 {
		panic("no return value specified for GetObjectMeta")
	}

	var r0 os.FileInfo
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) (os.FileInfo, error)); ok {
		return returnFunc(ctx, bucket, name)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) os.FileInfo); ok {
		r0 = returnFunc(ctx, bucket, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(os.FileInfo)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = returnFunc(ctx, bucket, name)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockObjectManager_GetObjectMeta_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetObjectMeta'
type MockObjectManager_GetObjectMeta_Call struct {
	*mock.Call
}

// GetObjectMeta is a helper method to define mock.On call
//   - ctx
//   - bucket
//   - name
func (_e *MockObjectManager_Expecter) GetObjectMeta(ctx interface{}, bucket interface{}, name interface{}) *MockObjectManager_GetObjectMeta_Call {
	return &MockObjectManager_GetObjectMeta_Call{Call: _e.mock.On("GetObjectMeta", ctx, bucket, name)}
}

func (_c *MockObjectManager_GetObjectMeta_Call) Run(run func(ctx context.Context, bucket string, name string)) *MockObjectManager_GetObjectMeta_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockObjectManager_GetObjectMeta_Call) Return(v os.FileInfo, err error) *MockObjectManager_GetObjectMeta_Call {
	_c.Call.Return(v, err)
	return _c
}

func (_c *MockObjectManager_GetObjectMeta_Call) RunAndReturn(run func(ctx context.Context, bucket string, name string) (os.FileInfo, error)) *MockObjectManager_GetObjectMeta_Call {
	_c.Call.Return(run)
	return _c
}

// GetObjectPart provides a mock function for the type MockObjectManager
func (_mock *MockObjectManager) GetObjectPart(ctx context.Context, bucket string, name string, start int64, end int64) (io.Reader, utils.CleanUp, error) {
	ret := _mock.Called(ctx, bucket, name, start, end)

	if len(ret) == 0 {
		panic("no return value specified for GetObjectPart")
	}

	var r0 io.Reader
	var r1 utils.CleanUp
	var r2 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, int64, int64) (io.Reader, utils.CleanUp, error)); ok {
		return returnFunc(ctx, bucket, name, start, end)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, int64, int64) io.Reader); ok {
		r0 = returnFunc(ctx, bucket, name, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.Reader)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, string, int64, int64) utils.CleanUp); ok {
		r1 = returnFunc(ctx, bucket, name, start, end)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(utils.CleanUp)
		}
	}
	if returnFunc, ok := ret.Get(2).(func(context.Context, string, string, int64, int64) error); ok {
		r2 = returnFunc(ctx, bucket, name, start, end)
	} else {
		r2 = ret.Error(2)
	}
	return r0, r1, r2
}

// MockObjectManager_GetObjectPart_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetObjectPart'
type MockObjectManager_GetObjectPart_Call struct {
	*mock.Call
}

// GetObjectPart is a helper method to define mock.On call
//   - ctx
//   - bucket
//   - name
//   - start
//   - end
func (_e *MockObjectManager_Expecter) GetObjectPart(ctx interface{}, bucket interface{}, name interface{}, start interface{}, end interface{}) *MockObjectManager_GetObjectPart_Call {
	return &MockObjectManager_GetObjectPart_Call{Call: _e.mock.On("GetObjectPart", ctx, bucket, name, start, end)}
}

func (_c *MockObjectManager_GetObjectPart_Call) Run(run func(ctx context.Context, bucket string, name string, start int64, end int64)) *MockObjectManager_GetObjectPart_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(int64), args[4].(int64))
	})
	return _c
}

func (_c *MockObjectManager_GetObjectPart_Call) Return(reader io.Reader, cleanUp utils.CleanUp, err error) *MockObjectManager_GetObjectPart_Call {
	_c.Call.Return(reader, cleanUp, err)
	return _c
}

func (_c *MockObjectManager_GetObjectPart_Call) RunAndReturn(run func(ctx context.Context, bucket string, name string, start int64, end int64) (io.Reader, utils.CleanUp, error)) *MockObjectManager_GetObjectPart_Call {
	_c.Call.Return(run)
	return _c
}

// IsObjectExist provides a mock function for the type MockObjectManager
func (_mock *MockObjectManager) IsObjectExist(ctx context.Context, bucket string, name string) (bool, error) {
	ret := _mock.Called(ctx, bucket, name)

	if len(ret) == 0 {
		panic("no return value specified for IsObjectExist")
	}

	var r0 bool
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) (bool, error)); ok {
		return returnFunc(ctx, bucket, name)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) bool); ok {
		r0 = returnFunc(ctx, bucket, name)
	} else {
		r0 = ret.Get(0).(bool)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = returnFunc(ctx, bucket, name)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockObjectManager_IsObjectExist_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsObjectExist'
type MockObjectManager_IsObjectExist_Call struct {
	*mock.Call
}

// IsObjectExist is a helper method to define mock.On call
//   - ctx
//   - bucket
//   - name
func (_e *MockObjectManager_Expecter) IsObjectExist(ctx interface{}, bucket interface{}, name interface{}) *MockObjectManager_IsObjectExist_Call {
	return &MockObjectManager_IsObjectExist_Call{Call: _e.mock.On("IsObjectExist", ctx, bucket, name)}
}

func (_c *MockObjectManager_IsObjectExist_Call) Run(run func(ctx context.Context, bucket string, name string)) *MockObjectManager_IsObjectExist_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockObjectManager_IsObjectExist_Call) Return(b bool, err error) *MockObjectManager_IsObjectExist_Call {
	_c.Call.Return(b, err)
	return _c
}

func (_c *MockObjectManager_IsObjectExist_Call) RunAndReturn(run func(ctx context.Context, bucket string, name string) (bool, error)) *MockObjectManager_IsObjectExist_Call {
	_c.Call.Return(run)
	return _c
}

// ListAllObjects provides a mock function for the type MockObjectManager
func (_mock *MockObjectManager) ListAllObjects(ctx context.Context, bucket string, prefix string) ([]os.FileInfo, error) {
	ret := _mock.Called(ctx, bucket, prefix)

	if len(ret) == 0 {
		panic("no return value specified for ListAllObjects")
	}

	var r0 []os.FileInfo
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) ([]os.FileInfo, error)); ok {
		return returnFunc(ctx, bucket, prefix)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) []os.FileInfo); ok {
		r0 = returnFunc(ctx, bucket, prefix)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]os.FileInfo)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = returnFunc(ctx, bucket, prefix)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockObjectManager_ListAllObjects_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListAllObjects'
type MockObjectManager_ListAllObjects_Call struct {
	*mock.Call
}

// ListAllObjects is a helper method to define mock.On call
//   - ctx
//   - bucket
//   - prefix
func (_e *MockObjectManager_Expecter) ListAllObjects(ctx interface{}, bucket interface{}, prefix interface{}) *MockObjectManager_ListAllObjects_Call {
	return &MockObjectManager_ListAllObjects_Call{Call: _e.mock.On("ListAllObjects", ctx, bucket, prefix)}
}

func (_c *MockObjectManager_ListAllObjects_Call) Run(run func(ctx context.Context, bucket string, prefix string)) *MockObjectManager_ListAllObjects_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockObjectManager_ListAllObjects_Call) Return(vs []os.FileInfo, err error) *MockObjectManager_ListAllObjects_Call {
	_c.Call.Return(vs, err)
	return _c
}

func (_c *MockObjectManager_ListAllObjects_Call) RunAndReturn(run func(ctx context.Context, bucket string, prefix string) ([]os.FileInfo, error)) *MockObjectManager_ListAllObjects_Call {
	_c.Call.Return(run)
	return _c
}

// ListObjects provides a mock function for the type MockObjectManager
func (_mock *MockObjectManager) ListObjects(ctx context.Context, bucket string, prefix string, count int) ([]os.FileInfo, error) {
	ret := _mock.Called(ctx, bucket, prefix, count)

	if len(ret) == 0 {
		panic("no return value specified for ListObjects")
	}

	var r0 []os.FileInfo
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, int) ([]os.FileInfo, error)); ok {
		return returnFunc(ctx, bucket, prefix, count)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, int) []os.FileInfo); ok {
		r0 = returnFunc(ctx, bucket, prefix, count)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]os.FileInfo)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, string, int) error); ok {
		r1 = returnFunc(ctx, bucket, prefix, count)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockObjectManager_ListObjects_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListObjects'
type MockObjectManager_ListObjects_Call struct {
	*mock.Call
}

// ListObjects is a helper method to define mock.On call
//   - ctx
//   - bucket
//   - prefix
//   - count
func (_e *MockObjectManager_Expecter) ListObjects(ctx interface{}, bucket interface{}, prefix interface{}, count interface{}) *MockObjectManager_ListObjects_Call {
	return &MockObjectManager_ListObjects_Call{Call: _e.mock.On("ListObjects", ctx, bucket, prefix, count)}
}

func (_c *MockObjectManager_ListObjects_Call) Run(run func(ctx context.Context, bucket string, prefix string, count int)) *MockObjectManager_ListObjects_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(int))
	})
	return _c
}

func (_c *MockObjectManager_ListObjects_Call) Return(vs []os.FileInfo, err error) *MockObjectManager_ListObjects_Call {
	_c.Call.Return(vs, err)
	return _c
}

func (_c *MockObjectManager_ListObjects_Call) RunAndReturn(run func(ctx context.Context, bucket string, prefix string, count int) ([]os.FileInfo, error)) *MockObjectManager_ListObjects_Call {
	_c.Call.Return(run)
	return _c
}

// PutObject provides a mock function for the type MockObjectManager
func (_mock *MockObjectManager) PutObject(ctx context.Context, bucket string, name string, reader io.Reader) (bool, error) {
	ret := _mock.Called(ctx, bucket, name, reader)

	if len(ret) == 0 {
		panic("no return value specified for PutObject")
	}

	var r0 bool
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, io.Reader) (bool, error)); ok {
		return returnFunc(ctx, bucket, name, reader)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, io.Reader) bool); ok {
		r0 = returnFunc(ctx, bucket, name, reader)
	} else {
		r0 = ret.Get(0).(bool)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, string, io.Reader) error); ok {
		r1 = returnFunc(ctx, bucket, name, reader)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockObjectManager_PutObject_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PutObject'
type MockObjectManager_PutObject_Call struct {
	*mock.Call
}

// PutObject is a helper method to define mock.On call
//   - ctx
//   - bucket
//   - name
//   - reader
func (_e *MockObjectManager_Expecter) PutObject(ctx interface{}, bucket interface{}, name interface{}, reader interface{}) *MockObjectManager_PutObject_Call {
	return &MockObjectManager_PutObject_Call{Call: _e.mock.On("PutObject", ctx, bucket, name, reader)}
}

func (_c *MockObjectManager_PutObject_Call) Run(run func(ctx context.Context, bucket string, name string, reader io.Reader)) *MockObjectManager_PutObject_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(io.Reader))
	})
	return _c
}

func (_c *MockObjectManager_PutObject_Call) Return(b bool, err error) *MockObjectManager_PutObject_Call {
	_c.Call.Return(b, err)
	return _c
}

func (_c *MockObjectManager_PutObject_Call) RunAndReturn(run func(ctx context.Context, bucket string, name string, reader io.Reader) (bool, error)) *MockObjectManager_PutObject_Call {
	_c.Call.Return(run)
	return _c
}
