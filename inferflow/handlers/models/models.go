package models

import (
	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/predator"
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
