package inmemorycache

import "github.com/stretchr/testify/mock"

type MockInMemoryCacheClient struct {
	mock.Mock
}

func (m *MockInMemoryCacheClient) MGet(keys []string, tags *[]string) map[string][]byte {
	args := m.Called(keys, tags)
	return args.Get(0).(map[string][]byte)
}

func (m *MockInMemoryCacheClient) MSet(data *map[string]interface{}, ttl int, tags *[]string) {
	m.Called(data, ttl, tags)
}
