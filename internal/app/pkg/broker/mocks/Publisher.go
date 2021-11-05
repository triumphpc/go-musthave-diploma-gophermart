// Code generated by mockery 2.9.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// Publisher is an autogenerated mock type for the Publisher type
type Publisher struct {
	mock.Mock
}

// Channel provides a mock function with given fields:
func (_m *Publisher) Channel() <-chan func() error {
	ret := _m.Called()

	var r0 <-chan func() error
	if rf, ok := ret.Get(0).(func() <-chan func() error); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan func() error)
		}
	}

	return r0
}

// Publish provides a mock function with given fields: task
func (_m *Publisher) Publish(task func() error) error {
	ret := _m.Called(task)

	var r0 error
	if rf, ok := ret.Get(0).(func(func() error) error); ok {
		r0 = rf(task)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
