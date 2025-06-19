package handler

import (
	"encoding/json"
	"errors"
	"testing"

	ofsConfig "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config"
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/handler/mocks"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteFeatures(t *testing.T) {
	tests := []struct {
		name                string
		request             *DeleteFeaturesRequest
		mockFeatureGroup    *ofsConfig.FeatureGroup
		mockFeatureGroupErr error
		mockCreateID        uint
		mockCreateErr       error
		expectedRequestID   uint
		expectedErr         string
	}{
		{
			name: "successfully records feature deletion request",
			request: &DeleteFeaturesRequest{
				UserId:            "user1",
				EntityLabel:       "entity",
				FeatureGroupLabel: "group",
				FeatureLabels:     []string{"feature1"},
			},
			mockFeatureGroup: &ofsConfig.FeatureGroup{
				ActiveVersion: "1",
				Features: map[string]ofsConfig.Feature{
					"1": {
						Labels:      "feature1,feature2",
						FeatureMeta: map[string]ofsConfig.FeatureMeta{"feature1": {}, "feature2": {}},
					},
				},
			},
			mockFeatureGroupErr: nil,
			mockCreateID:        101,
			mockCreateErr:       nil,
			expectedRequestID:   101,
			expectedErr:         "",
		},
		{
			name: "feature group not found error",
			request: &DeleteFeaturesRequest{
				EntityLabel:       "nonexistent_entity",
				FeatureGroupLabel: "nonexistent_group",
				FeatureLabels:     []string{"feature1"},
			},
			mockFeatureGroup:    nil,
			mockFeatureGroupErr: errors.New("feature group not found"),
			expectedRequestID:   0,
			expectedErr:         "feature group not found",
		},
		{
			name: "some features not found in feature group",
			request: &DeleteFeaturesRequest{
				EntityLabel:       "entity",
				FeatureGroupLabel: "group",
				FeatureLabels:     []string{"feature1", "feature3"},
			},
			mockFeatureGroup: &ofsConfig.FeatureGroup{
				ActiveVersion: "1",
				Features: map[string]ofsConfig.Feature{
					"1": {
						Labels:      "feature1,feature2",
						FeatureMeta: map[string]ofsConfig.FeatureMeta{"feature1": {}, "feature2": {}},
					},
				},
			},
			mockFeatureGroupErr: nil,
			expectedRequestID:   0,
			expectedErr:         "features not found in the active version schema: feature3",
		},
		{
			name: "multiple features not found",
			request: &DeleteFeaturesRequest{
				EntityLabel:       "entity",
				FeatureGroupLabel: "group",
				FeatureLabels:     []string{"feature1", "feature3", "feature4"},
			},
			mockFeatureGroup: &ofsConfig.FeatureGroup{
				ActiveVersion: "1",
				Features: map[string]ofsConfig.Feature{
					"1": {
						Labels:      "feature1,feature2",
						FeatureMeta: map[string]ofsConfig.FeatureMeta{"feature1": {}, "feature2": {}},
					},
				},
			},
			mockFeatureGroupErr: nil,
			expectedRequestID:   0,
			expectedErr:         "features not found in the active version schema: feature3, feature4",
		},
		{
			name: "error creating delete request in repository",
			request: &DeleteFeaturesRequest{
				UserId:            "user1",
				EntityLabel:       "entity",
				FeatureGroupLabel: "group",
				FeatureLabels:     []string{"feature1"},
			},
			mockFeatureGroup: &ofsConfig.FeatureGroup{
				ActiveVersion: "1",
				Features: map[string]ofsConfig.Feature{
					"1": {
						Labels:      "feature1,feature2",
						FeatureMeta: map[string]ofsConfig.FeatureMeta{"feature1": {}, "feature2": {}},
					},
				},
			},
			mockFeatureGroupErr: nil,
			mockCreateID:        0,
			mockCreateErr:       errors.New("repository create error"),
			expectedRequestID:   0,
			expectedErr:         "repository create error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig := new(mocks.ConfigManager)
			mockFeatureRepo := new(mocks.FeatureRepository)

			mockConfig.On("GetFeatureGroup", tt.request.EntityLabel, tt.request.FeatureGroupLabel).
				Return(tt.mockFeatureGroup, tt.mockFeatureGroupErr)

			shouldMockFeatureRepoCreate := false
			if tt.mockFeatureGroupErr == nil && tt.mockFeatureGroup != nil {
				allFeaturesFound := true
				if tt.mockFeatureGroup.Features != nil && tt.mockFeatureGroup.Features[tt.mockFeatureGroup.ActiveVersion].FeatureMeta != nil {
					for _, label := range tt.request.FeatureLabels {
						if _, exists := tt.mockFeatureGroup.Features[tt.mockFeatureGroup.ActiveVersion].FeatureMeta[label]; !exists {
							allFeaturesFound = false
							break
						}
					}
				} else {
					if len(tt.request.FeatureLabels) > 0 {
						allFeaturesFound = false
					}
				}
				shouldMockFeatureRepoCreate = allFeaturesFound
			}

			if shouldMockFeatureRepoCreate {
				mockFeatureRepo.On("Create", mock.AnythingOfType("*features.Table")).
					Return(int(tt.mockCreateID), tt.mockCreateErr)
			}

			store := &OnlineFeatureStore{
				Config:      mockConfig,
				featureRepo: mockFeatureRepo,
			}

			requestID, err := store.DeleteFeatures(tt.request)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRequestID, requestID)
			}

			mockConfig.AssertExpectations(t)
			mockFeatureRepo.AssertExpectations(t)
		})
	}
}

func TestProcessDeleteFeatures(t *testing.T) {
	tests := []struct {
		name                string
		request             *ProcessDeleteFeaturesRequest
		mockFeature         *features.Table
		mockFeatureErr      error
		mockFeatureGroup    *ofsConfig.FeatureGroup
		mockFeatureGroupErr error
		mockDeleteErr       error
		mockUpdateErr       error
		expectedErr         string
	}{
		{
			name: "successfully processes feature deletion request",
			request: &ProcessDeleteFeaturesRequest{
				ApproverId: "admin",
				Role:       "admin",
				RequestId:  101,
				Status:     "APPROVED",
			},
			mockFeature: &features.Table{
				RequestId:         101,
				Status:            "PENDING APPROVAL",
				RequestType:       "DELETE",
				EntityLabel:       "entity",
				FeatureGroupLabel: "group",
				Payload:           []byte(`{"entity_label":"entity","feature_group_label":"group","feature_labels":["feature1"]}`),
			},
			mockFeatureErr: nil,
			mockFeatureGroup: &ofsConfig.FeatureGroup{
				ActiveVersion: "1",
				Features: map[string]ofsConfig.Feature{
					"1": {
						Labels:      "feature1,feature2",
						FeatureMeta: map[string]ofsConfig.FeatureMeta{"feature1": {}, "feature2": {}},
					},
				},
			},
			mockFeatureGroupErr: nil,
			mockDeleteErr:       nil,
			mockUpdateErr:       nil,
			expectedErr:         "",
		},
		{
			name: "feature request not found",
			request: &ProcessDeleteFeaturesRequest{
				RequestId: 101,
			},
			mockFeature:    nil,
			mockFeatureErr: errors.New("feature request not found"),
			expectedErr:    "feature request not found",
		},
		{
			name: "feature request already processed",
			request: &ProcessDeleteFeaturesRequest{
				RequestId: 101,
			},
			mockFeature: &features.Table{
				RequestId: 101,
				Status:    "APPROVED",
			},
			mockFeatureErr: nil,
			expectedErr:    "delete features request is already Processed for request id 101",
		},
		{
			name: "reject feature deletion request",
			request: &ProcessDeleteFeaturesRequest{
				RequestId:    101,
				Status:       "REJECTED",
				RejectReason: "some reason",
			},
			mockFeature: &features.Table{
				RequestId: 101,
				Status:    "PENDING APPROVAL",
			},
			mockFeatureErr: nil,
			mockUpdateErr:  nil,
			expectedErr:    "",
		},
		{
			name: "error updating status on rejection",
			request: &ProcessDeleteFeaturesRequest{
				RequestId: 101,
				Status:    "REJECTED",
			},
			mockFeature: &features.Table{
				RequestId: 101,
				Status:    "PENDING APPROVAL",
			},
			mockFeatureErr: nil,
			mockUpdateErr:  errors.New("db update error"),
			expectedErr:    "db update error",
		},
		{
			name: "processing approved request with feature group not found",
			request: &ProcessDeleteFeaturesRequest{
				RequestId: 101,
				Status:    "APPROVED",
			},
			mockFeature: &features.Table{
				RequestId: 101,
				Status:    "PENDING APPROVAL",
				Payload:   []byte(`{"entity_label":"entity","feature_group_label":"group","feature_labels":["feature1"]}`),
			},
			mockFeatureErr:      nil,
			mockFeatureGroup:    nil,
			mockFeatureGroupErr: errors.New("feature group not found"),
			expectedErr:         "feature group not found",
		},
		{
			name: "error from config manager on delete",
			request: &ProcessDeleteFeaturesRequest{
				RequestId: 101,
				Status:    "APPROVED",
			},
			mockFeature: &features.Table{
				RequestId: 101,
				Status:    "PENDING APPROVAL",
				Payload:   []byte(`{"entity_label":"entity","feature_group_label":"group","feature_labels":["feature1"]}`),
			},
			mockFeatureErr: nil,
			mockFeatureGroup: &ofsConfig.FeatureGroup{
				ActiveVersion: "1",
				Features: map[string]ofsConfig.Feature{
					"1": {FeatureMeta: map[string]ofsConfig.FeatureMeta{"feature1": {}}},
				},
			},
			mockFeatureGroupErr: nil,
			mockDeleteErr:       errors.New("config delete error"),
			expectedErr:         "config delete error",
		},
		{
			name: "error updating status on successful approval",
			request: &ProcessDeleteFeaturesRequest{
				RequestId: 101,
				Status:    "APPROVED",
			},
			mockFeature: &features.Table{
				RequestId: 101,
				Status:    "PENDING APPROVAL",
				Payload:   []byte(`{"entity_label":"entity","feature_group_label":"group","feature_labels":["feature1"]}`),
			},
			mockFeatureErr: nil,
			mockFeatureGroup: &ofsConfig.FeatureGroup{
				ActiveVersion: "1",
				Features: map[string]ofsConfig.Feature{
					"1": {FeatureMeta: map[string]ofsConfig.FeatureMeta{"feature1": {}}},
				},
			},
			mockFeatureGroupErr: nil,
			mockDeleteErr:       nil,
			mockUpdateErr:       errors.New("db update error"),
			expectedErr:         "db update error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig := new(mocks.ConfigManager)
			mockFeatureRepo := new(mocks.FeatureRepository)

			mockFeatureRepo.On("GetById", tt.request.RequestId).Return(tt.mockFeature, tt.mockFeatureErr)

			if tt.mockFeatureErr == nil && tt.mockFeature != nil && tt.mockFeature.Status == "PENDING APPROVAL" {
				if tt.request.Status == "REJECTED" {
					mockFeatureRepo.On("Update", mock.AnythingOfType("*features.Table")).Return(tt.mockUpdateErr)
				} else {
					var deletePayload DeleteFeaturesRequest
					if tt.mockFeature.Payload != nil {
						_ = json.Unmarshal(tt.mockFeature.Payload, &deletePayload)
					}

					mockConfig.On("GetFeatureGroup", deletePayload.EntityLabel, deletePayload.FeatureGroupLabel).Return(tt.mockFeatureGroup, tt.mockFeatureGroupErr)

					if tt.mockFeatureGroupErr == nil && tt.mockFeatureGroup != nil {
						allFeaturesFound := true
						if tt.mockFeatureGroup.Features[tt.mockFeatureGroup.ActiveVersion].FeatureMeta != nil {
							for _, label := range deletePayload.FeatureLabels {
								if _, exists := tt.mockFeatureGroup.Features[tt.mockFeatureGroup.ActiveVersion].FeatureMeta[label]; !exists {
									allFeaturesFound = false
									break
								}
							}
						} else if len(deletePayload.FeatureLabels) > 0 {
							allFeaturesFound = false
						}

						if allFeaturesFound {
							mockConfig.On("DeleteFeatures", deletePayload.EntityLabel, deletePayload.FeatureGroupLabel, deletePayload.FeatureLabels).Return(tt.mockDeleteErr)
							if tt.mockDeleteErr == nil {
								mockFeatureRepo.On("Update", mock.AnythingOfType("*features.Table")).Return(tt.mockUpdateErr)
							}
						}
					}
				}
			}

			store := &OnlineFeatureStore{
				Config:      mockConfig,
				featureRepo: mockFeatureRepo,
			}

			err := store.ProcessDeleteFeatures(tt.request)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			mockConfig.AssertExpectations(t)
			mockFeatureRepo.AssertExpectations(t)
		})
	}
}
