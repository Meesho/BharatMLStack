package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseToNormalizedEntities(t *testing.T) {
	e := &Etcd{}

	testCases := []struct {
		name     string
		entities map[string]Entity
		want     *NormalizedEntities
		wantErr  bool
	}{
		{
			name: "successful parsing",
			entities: map[string]Entity{
				"user": {
					Label: "user",
					Keys: map[string]Key{
						"0": {EntityLabel: "user_id", ColumnLabel: "id", Sequence: 0},
					},
					FeatureGroups: map[string]FeatureGroup{
						"demographics": {
							Id: 1,
							Columns: map[string]Column{
								"age": {Label: "seg_0"},
							},
							ActiveVersion: "1",
							Features: map[string]FeatureSchema{
								"1": {
									Labels: "age,gender",
									FeatureMeta: map[string]FeatureMeta{
										"age":    {Sequence: 0, DefaultValuesInBytes: []byte("25")},
										"gender": {Sequence: 1, DefaultValuesInBytes: []byte("M")},
									},
								},
							},
						},
					},
				},
			},
			want: &NormalizedEntities{
				Entities: map[string]NormalizedEntity{
					"user": {
						PrimaryKeyColumnNames: []string{"id"},
						ColumnToPKIdMap: map[string]string{
							"id": "user_id",
						},
						FGs: map[int]NormalizedFGConfig{
							1: {
								Versions: map[_version][]string{
									1: {"age", "gender"},
								},
								ActiveVersion: 1,
								DefaultsInBytes: map[_version]map[string][]byte{
									1: {
										"age":    []byte("25"),
										"gender": []byte("M"),
									},
								},
								Sequences: map[_version]map[string]int{
									1: {
										"age":    0,
										"gender": 1,
									},
								},
								StringLengths: map[_version][]uint16{
									1: {0, 0},
								},
								VectorLengths: map[_version][]uint16{
									1: {0, 0},
								},
								NumofFeatures: map[_version]int{
									1: 2,
								},
								Columns: []string{"seg_0"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid version number",
			entities: map[string]Entity{
				"user": {
					Label: "user",
					Keys: map[string]Key{
						"0": {EntityLabel: "user_id", ColumnLabel: "id", Sequence: 0},
					},
					FeatureGroups: map[string]FeatureGroup{
						"demographics": {
							Features: map[string]FeatureSchema{
								"invalid": {}, // Invalid version number
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "multiple keys parsing",
			entities: map[string]Entity{
				"query_x_catalog": {
					Label: "query_x_catalog",
					Keys: map[string]Key{
						"query_id":   {EntityLabel: "query_id", ColumnLabel: "qid", Sequence: 0},
						"catalog_id": {EntityLabel: "catalog_id", ColumnLabel: "cid", Sequence: 1},
					},
					FeatureGroups: map[string]FeatureGroup{
						"relevance": {
							Id:            2,
							ActiveVersion: "1",
							Columns: map[string]Column{
								"score": {Label: "rel_score"},
								"rank":  {Label: "position"},
							},
							Features: map[string]FeatureSchema{
								"1": {
									Labels: "score,rank",
									FeatureMeta: map[string]FeatureMeta{
										"score": {Sequence: 0, DefaultValuesInBytes: []byte("0.0")},
										"rank":  {Sequence: 1, DefaultValuesInBytes: []byte("999")},
									},
								},
							},
						},
					},
				},
			},
			want: &NormalizedEntities{
				Entities: map[string]NormalizedEntity{
					"query_x_catalog": {
						PrimaryKeyColumnNames: []string{"qid", "cid"},
						ColumnToPKIdMap: map[string]string{
							"qid": "query_id",
							"cid": "catalog_id",
						},
						FGs: map[int]NormalizedFGConfig{
							2: {
								Versions: map[_version][]string{
									1: {"score", "rank"},
								},
								DefaultsInBytes: map[_version]map[string][]byte{
									1: {
										"score": []byte("0.0"),
										"rank":  []byte("999"),
									},
								},
								Sequences: map[_version]map[string]int{
									1: {
										"score": 0,
										"rank":  1,
									},
								},
								Columns: []string{"position", "rel_score"},
								StringLengths: map[_version][]uint16{
									1: {0, 0},
								},
								VectorLengths: map[_version][]uint16{
									1: {0, 0},
								},
								NumofFeatures: map[_version]int{
									1: 2,
								},
								ActiveVersion: 1,
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.parseToNormalizedEntities(tc.entities)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
