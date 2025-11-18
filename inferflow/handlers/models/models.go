package models

import (
	"github.com/Meesho/helix-clients/pkg/clients/predator"
	clientmodels "github.com/Meesho/price-aggregator-go/pricingfeatureretrieval/client/models"
)

type FeatureComponentBuilder struct {
	Labels                 []string
	UniqueEntityIds        []string
	EntitiesToKeyValuesMap map[string][]string
	RequestedEntityIds     []string
	Result                 [][][]byte
	LabelIndex             map[string]int
	UniqueEntityIndex      map[string]int
	KeySchema              []string
	KeyValues              [][]string
	InMemPresent           []bool
	InMemoryMissCount      int
}

type PredatorComponentBuilder struct {
	StringToByteCache map[string][]byte
	FeaturePayloads   [][][][]byte
	Inputs            []predator.Input
	Outputs           []predator.Output
	Scores            [][][]byte
	ModelScoreCount   int
}

type NumerixComponentBuilder struct {
	Matrix        [][][]byte
	MatrixColumns []string
	Schema        []string
	Scores        [][]byte
}

type RealTimePricingFeatureBuilder struct {
	Labels                 []string
	UniqueEntityIds        []string
	EntitiesToKeyValuesMap map[string][]string
	RequestedEntityIds     []string
	Result                 [][][]byte
	LabelIndex             map[string]int
	UniqueEntityIndex      map[string]int
	KeySchema              []string
	KeyValues              [][]string
	InMemPresent           []bool
	InMemoryMissCount      int
	IdMap                  []map[interface{}][]string
	EntityIdsMap           map[string]clientmodels.EntityId
}
