// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"
import v1 "k8s.io/api/core/v1"

// NamespaceCreator is an autogenerated mock type for the NamespaceCreator type
type NamespaceCreator struct {
	mock.Mock
}

// CreateNamespace provides a mock function with given fields: namespace
func (_m *NamespaceCreator) CreateNamespace(namespace string) (*v1.Namespace, error) {
	ret := _m.Called(namespace)

	var r0 *v1.Namespace
	if rf, ok := ret.Get(0).(func(string) *v1.Namespace); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Namespace)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}