package vector

import "github.com/stretchr/testify/mock"

// Ensure MockDatabase implements Database interface
var _ Database = (*MockDatabase)(nil)

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) CreateCollection(entity string, model string, variant string, version int) error {
	args := m.Called(entity, model, variant, version)
	return args.Error(0)
}

func (m *MockDatabase) BulkUpsert(upsertRequest UpsertRequest) error {
	args := m.Called(upsertRequest)
	return args.Error(0)
}

func (m *MockDatabase) BulkDelete(deleteRequest DeleteRequest) error {
	args := m.Called(deleteRequest)
	return args.Error(0)
}

func (m *MockDatabase) BulkUpsertPayload(upsertPayloadRequest UpsertPayloadRequest) error {
	args := m.Called(upsertPayloadRequest)
	return args.Error(0)
}

func (m *MockDatabase) DeleteCollection(entity, model, variant string, version int) error {
	args := m.Called(entity, model, variant, version)
	return args.Error(0)
}

func (m *MockDatabase) UpdateIndexingThreshold(entity, model, variant string, version int, indexingThreshold string) error {
	args := m.Called(entity, model, variant, version, indexingThreshold)
	return args.Error(0)
}

func (m *MockDatabase) GetCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error) {
	args := m.Called(entity, model, variant, version)
	if err := args.Error(1); err != nil {
		return nil, err
	}
	return args.Get(0).(*CollectionInfoResponse), nil
}

func (m *MockDatabase) GetReadCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error) {
	args := m.Called(entity, model, variant, version)
	if err := args.Error(1); err != nil {
		return nil, err
	}
	return args.Get(0).(*CollectionInfoResponse), nil
}

func (m *MockDatabase) RefreshClients(key, value, eventType string) error {
	args := m.Called(key, value, eventType)
	return args.Error(0)
}

func (m *MockDatabase) CreateFieldIndexes(entity, model, variant string, version int) error {
	args := m.Called(entity, model, variant, version)
	return args.Error(0)
}

func (m *MockDatabase) BatchQuery(bulkRequest *BatchQueryRequest, metricTags []string) (*BatchQueryResponse, error) {
	args := m.Called(bulkRequest, metricTags)
	if err := args.Error(1); err != nil {
		return nil, err
	}
	return args.Get(0).(*BatchQueryResponse), nil
}

// SetTestInstance sets the package-level vectorDb singleton to the given mock.
// Use only in tests.
func SetTestInstance(db Database) {
	vectorDb = db
}

// ResetTestInstance clears the package-level vectorDb singleton.
// Use only in tests.
func ResetTestInstance() {
	vectorDb = nil
}
