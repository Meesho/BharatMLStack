package embedding

import "github.com/stretchr/testify/mock"

type MockStore struct {
	mock.Mock
}

func (m *MockStore) BulkQuery(storeId string, bulkQuery *BulkQuery) error {
	args := m.Called(storeId, bulkQuery)
	return args.Error(0)
}

func (m *MockStore) Persist(storeId string, ttl int, payload Payload) error {
	args := m.Called(storeId, ttl, payload)
	if args.Get(0) == nil {
		return args.Error(0)
	}
	return nil
}
