// Code generated by mockery 2.9.4. DO NOT EDIT.

package influxdb

import (
	context "context"

	api "github.com/influxdata/influxdb-client-go/v2/api"

	domain "github.com/influxdata/influxdb-client-go/v2/domain"

	http "github.com/influxdata/influxdb-client-go/v2/api/http"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	mock "github.com/stretchr/testify/mock"
)

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

// AuthorizationsAPI provides a mock function with given fields:
func (_m *MockClient) AuthorizationsAPI() api.AuthorizationsAPI {
	ret := _m.Called()

	var r0 api.AuthorizationsAPI
	if rf, ok := ret.Get(0).(func() api.AuthorizationsAPI); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.AuthorizationsAPI)
		}
	}

	return r0
}

// BucketsAPI provides a mock function with given fields:
func (_m *MockClient) BucketsAPI() api.BucketsAPI {
	ret := _m.Called()

	var r0 api.BucketsAPI
	if rf, ok := ret.Get(0).(func() api.BucketsAPI); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.BucketsAPI)
		}
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *MockClient) Close() {
	_m.Called()
}

// DeleteAPI provides a mock function with given fields:
func (_m *MockClient) DeleteAPI() api.DeleteAPI {
	ret := _m.Called()

	var r0 api.DeleteAPI
	if rf, ok := ret.Get(0).(func() api.DeleteAPI); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.DeleteAPI)
		}
	}

	return r0
}

// GetMoisture provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockClient) GetMoisture(_a0 context.Context, _a1 int, _a2 string) (float64, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 float64
	if rf, ok := ret.Get(0).(func(context.Context, int, string) float64); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Get(0).(float64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int, string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HTTPService provides a mock function with given fields:
func (_m *MockClient) HTTPService() http.Service {
	ret := _m.Called()

	var r0 http.Service
	if rf, ok := ret.Get(0).(func() http.Service); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(http.Service)
		}
	}

	return r0
}

// Health provides a mock function with given fields: ctx
func (_m *MockClient) Health(ctx context.Context) (*domain.HealthCheck, error) {
	ret := _m.Called(ctx)

	var r0 *domain.HealthCheck
	if rf, ok := ret.Get(0).(func(context.Context) *domain.HealthCheck); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.HealthCheck)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LabelsAPI provides a mock function with given fields:
func (_m *MockClient) LabelsAPI() api.LabelsAPI {
	ret := _m.Called()

	var r0 api.LabelsAPI
	if rf, ok := ret.Get(0).(func() api.LabelsAPI); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.LabelsAPI)
		}
	}

	return r0
}

// Options provides a mock function with given fields:
func (_m *MockClient) Options() *influxdb2.Options {
	ret := _m.Called()

	var r0 *influxdb2.Options
	if rf, ok := ret.Get(0).(func() *influxdb2.Options); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*influxdb2.Options)
		}
	}

	return r0
}

// OrganizationsAPI provides a mock function with given fields:
func (_m *MockClient) OrganizationsAPI() api.OrganizationsAPI {
	ret := _m.Called()

	var r0 api.OrganizationsAPI
	if rf, ok := ret.Get(0).(func() api.OrganizationsAPI); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.OrganizationsAPI)
		}
	}

	return r0
}

// QueryAPI provides a mock function with given fields: org
func (_m *MockClient) QueryAPI(org string) api.QueryAPI {
	ret := _m.Called(org)

	var r0 api.QueryAPI
	if rf, ok := ret.Get(0).(func(string) api.QueryAPI); ok {
		r0 = rf(org)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.QueryAPI)
		}
	}

	return r0
}

// Ready provides a mock function with given fields: ctx
func (_m *MockClient) Ready(ctx context.Context) (bool, error) {
	ret := _m.Called(ctx)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context) bool); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ServerURL provides a mock function with given fields:
func (_m *MockClient) ServerURL() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Setup provides a mock function with given fields: ctx, username, password, org, bucket, retentionPeriodHours
func (_m *MockClient) Setup(ctx context.Context, username string, password string, org string, bucket string, retentionPeriodHours int) (*domain.OnboardingResponse, error) {
	ret := _m.Called(ctx, username, password, org, bucket, retentionPeriodHours)

	var r0 *domain.OnboardingResponse
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string, int) *domain.OnboardingResponse); ok {
		r0 = rf(ctx, username, password, org, bucket, retentionPeriodHours)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.OnboardingResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, string, int) error); ok {
		r1 = rf(ctx, username, password, org, bucket, retentionPeriodHours)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TasksAPI provides a mock function with given fields:
func (_m *MockClient) TasksAPI() api.TasksAPI {
	ret := _m.Called()

	var r0 api.TasksAPI
	if rf, ok := ret.Get(0).(func() api.TasksAPI); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.TasksAPI)
		}
	}

	return r0
}

// UsersAPI provides a mock function with given fields:
func (_m *MockClient) UsersAPI() api.UsersAPI {
	ret := _m.Called()

	var r0 api.UsersAPI
	if rf, ok := ret.Get(0).(func() api.UsersAPI); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.UsersAPI)
		}
	}

	return r0
}

// WriteAPI provides a mock function with given fields: org, bucket
func (_m *MockClient) WriteAPI(org string, bucket string) api.WriteAPI {
	ret := _m.Called(org, bucket)

	var r0 api.WriteAPI
	if rf, ok := ret.Get(0).(func(string, string) api.WriteAPI); ok {
		r0 = rf(org, bucket)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.WriteAPI)
		}
	}

	return r0
}

// WriteAPIBlocking provides a mock function with given fields: org, bucket
func (_m *MockClient) WriteAPIBlocking(org string, bucket string) api.WriteAPIBlocking {
	ret := _m.Called(org, bucket)

	var r0 api.WriteAPIBlocking
	if rf, ok := ret.Get(0).(func(string, string) api.WriteAPIBlocking); ok {
		r0 = rf(org, bucket)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.WriteAPIBlocking)
		}
	}

	return r0
}
