package inmemorycache

import "github.com/stretchr/testify/mock"

type MockInMemoryCacheClient struct {
	mock.Mock
}

func (m *MockInMemoryCacheClient) Get(key []byte) ([]byte, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), nil
}

func (m *MockInMemoryCacheClient) Set(key, value []byte) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockInMemoryCacheClient) SetEx(key, value []byte, expiry int) error {
	args := m.Called(key, value, expiry)
	return args.Error(0)
}

func (m *MockInMemoryCacheClient) Delete(key []byte) bool {
	args := m.Called(key)
	return args.Get(0).(bool)
}
