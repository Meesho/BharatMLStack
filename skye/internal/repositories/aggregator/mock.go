package aggregator

import "github.com/stretchr/testify/mock"

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Query(storeId string, query *Query) (map[string]interface{}, error) {
	args := m.Called(storeId, query)
	if args.Get(0) == nil {
		return nil, args.Error(0)
	}
	return args.Get(0).(map[string]interface{}), nil
}

func (m *MockDatabase) Persist(storeId string, candidateId string, columns map[string]interface{}) error {
	args := m.Called(storeId, candidateId, columns)
	if args.Get(0) == nil {
		return args.Error(0)
	}
	return nil
}
