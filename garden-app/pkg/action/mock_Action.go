// Code generated by mockery v2.9.4. DO NOT EDIT.

package action

import (
	pkg "github.com/calvinmclean/automated-garden/garden-app/pkg"
	mock "github.com/stretchr/testify/mock"
)

// MockAction is an autogenerated mock type for the Action type
type MockAction struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockAction) Execute(_a0 *pkg.Garden, _a1 *pkg.Zone, _a2 Scheduler) {
	_m.Called(_a0, _a1, _a2)
}