package feature

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/persist"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
)

type RetrieveData struct {
	ReqColumnCount                     int
	Query                              *retrieve.Query
	Result                             *retrieve.Result
	EntityLabel                        string
	AllFGIdToDataType                  map[int]types.DataType
	ReqInMemCachedFGIds                ds.Set[int]
	ReqDistCachedFGIds                 ds.Set[int]
	ReqDbFGIds                         ds.Set[int]
	AllDistCachedFGIds                 ds.Set[int]
	AllFGIdToStoreId                   map[int]string
	AllFGLabelToFGId                   map[string]int
	AllFGIdToFGLabel                   map[int]string
	ReqFGIdToFeatureLabels             map[int]ds.Set[string]
	ReqFGIdToFeatureLabelWithQuantType map[int]map[string]types.DataType
	ReqKeyToIdx                        map[string]int
	ReqFGIds                           ds.Set[int]
	ReqIdxToFgIdToDdb                  map[int]map[int]*blocks.DeserializedPSDB
}

type PersistData struct {
	Query                  *persist.Query
	EntityLabel            string
	AllFGIdToFgConf        map[int]FgConf
	AllFGLabelToFGId       map[string]int
	StoreIdToMaxColumnSize map[string]int
	IsAnyFGInMemCached     bool
	IsAnyFGDistCached      bool
	StoreIdToRows          map[string][]Row
}

type FgConf struct {
	DataType    types.DataType
	FeatureMeta map[string]config.FeatureMeta
}

type Row struct {
	PkMap      map[string]string
	FgIdToPsDb map[int]*blocks.PermStorageDataBlock
}
