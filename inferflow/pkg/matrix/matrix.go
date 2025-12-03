package matrix

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/external/featurestore"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/models"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/typeconverter"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"
	"github.com/rs/zerolog/log"
)

// DataTypeString //datatypestring -> datatypestring
type ComponentMatrix struct {
	StringColumnIndexMap map[string]Column
	ByteColumnIndexMap   map[string]Column
	Rows                 []Row
}

type Row struct {
	StringData []string
	ByteData   [][]byte
}

type Column struct {
	Name     string
	Index    int
	DataType string
}

func (m *ComponentMatrix) InitComponentMatrix(rowCount int, stringColumnIndexMap, byteColumnIndexMap map[string]Column) {
	stringColCount := len(stringColumnIndexMap)
	byteColCount := len(byteColumnIndexMap)

	rows := make([]Row, rowCount)

	// Preallocate big contiguous slices
	stringDataBlock := make([]string, rowCount*stringColCount)
	byteDataBlock := make([][]byte, rowCount*byteColCount)

	for i := 0; i < rowCount; i++ {
		stringStart := i * stringColCount
		byteStart := i * byteColCount

		rows[i] = Row{
			StringData: stringDataBlock[stringStart : stringStart+stringColCount],
			ByteData:   byteDataBlock[byteStart : byteStart+byteColCount],
		}
	}

	m.StringColumnIndexMap = stringColumnIndexMap
	m.ByteColumnIndexMap = byteColumnIndexMap
	m.Rows = rows
}

func (m *ComponentMatrix) Get(rowIndex int, columnName string) []byte {
	col, ok := m.ByteColumnIndexMap[columnName]
	if !ok || rowIndex >= len(m.Rows) {
		return nil
	}
	return m.Rows[rowIndex].ByteData[col.Index]
}

func (m *ComponentMatrix) Set(rowIndex int, columnName string, value []byte) {
	col, ok := m.ByteColumnIndexMap[columnName]
	if !ok || rowIndex >= len(m.Rows) {
		return
	}
	m.Rows[rowIndex].ByteData[col.Index] = value
}

func (m *ComponentMatrix) PopulateStringData(columnToPopulate string, values []string) {
	col, ok := m.StringColumnIndexMap[columnToPopulate]
	if !ok {
		return
	}
	for i := 0; i < len(m.Rows) && i < len(values); i++ {
		m.Rows[i].StringData[col.Index] = values[i]
	}
}

func (m *ComponentMatrix) PopulateStringDataForCompositeKey(columnToPopulate string, values [][]string) {
	col, ok := m.StringColumnIndexMap[columnToPopulate]
	if !ok {
		return
	}
	for i := 0; i < len(m.Rows) && i < len(values); i++ {
		m.Rows[i].StringData[col.Index] = strings.Join(values[i], featurestore.CacheKeySeparator)
	}
}

func (m *ComponentMatrix) PopulateByteData(columnToPopulate string, score [][]byte) {
	col, ok := m.ByteColumnIndexMap[columnToPopulate]
	if !ok {
		return
	}

	if len(score) == 1 {
		for i := 0; i < len(m.Rows); i++ {
			m.Rows[i].ByteData[col.Index] = score[0]
		}
		return
	}

	for i := 0; i < len(m.Rows) && i < len(score); i++ {
		m.Rows[i].ByteData[col.Index] = score[i]
	}
}

func (m *ComponentMatrix) PopulateStringDataFromSingleValue(columnToPopulate string, value string) {
	col, ok := m.StringColumnIndexMap[columnToPopulate]
	if !ok {
		return
	}
	for i := 0; i < len(m.Rows); i++ {
		m.Rows[i].StringData[col.Index] = value
	}
}

type KeyValues struct {
	Values [][]string
}

// InitializeFeatureMatrix fo this
func (m *ComponentMatrix) GetColumnValuesWithKey(fsKeySlice []config.FSKey) KeyValues {
	rowCount := len(m.Rows)
	colCount := len(fsKeySlice)
	var keyValues KeyValues
	keyValues.Values = make([][]string, rowCount)
	keyValueBufferArray := make([]string, rowCount*colCount)
	for i := 0; i < rowCount; i++ {
		keyValues.Values[i] = keyValueBufferArray[i*colCount : (i+1)*colCount]
	}
	for i, key := range fsKeySlice {
		if col, ok := m.StringColumnIndexMap[key.Column]; ok {
			for row := 0; row < rowCount; row++ {
				keyValues.Values[row][i] = m.Rows[row].StringData[col.Index]
			}
		}
	}
	return keyValues
}

func (m *ComponentMatrix) PopulateMatrixOfColumnSlice(numerixComponentBuilder *models.NumerixComponentBuilder) {
	for i, row := range m.Rows {
		for j, col := range numerixComponentBuilder.MatrixColumns {
			byteCol, hasByteData := m.ByteColumnIndexMap[col]
			stringCol, hasStringData := m.StringColumnIndexMap[col]
			switch {
			case hasByteData:
				if row.ByteData[byteCol.Index] != nil && len(row.ByteData[byteCol.Index]) > 0 {
					numerixComponentBuilder.Matrix[i][j] = row.ByteData[byteCol.Index]
				} else {
					numerixComponentBuilder.Matrix[i][j] = utils.GetDefaultValuesInBytes(byteCol.DataType)
				}
			case hasStringData:
				if row.StringData[stringCol.Index] != "" {
					numerixComponentBuilder.Matrix[i][j] = []byte(row.StringData[stringCol.Index])
				} else {
					numerixComponentBuilder.Matrix[i][j] = utils.GetDefaultValuesInBytes(stringCol.DataType)
				}
			}
		}
	}
}

type PredatorResult struct {
	ByteToByteConversionCount int
	FeaturePayloads           [][][][]byte
}

func (m *ComponentMatrix) GetMatrixOfColumnSliceInString(columns []string) [][]string {
	matrix := make([][]string, len(m.Rows)+1)
	matrix[0] = columns
	for i, row := range m.Rows {
		newRow := make([]string, len(columns))
		for j, col := range columns {
			byteCol, ok := m.ByteColumnIndexMap[col]
			var val string
			var err error
			if ok {
				val, err = typeconverter.BytesToString(row.ByteData[byteCol.Index], byteCol.DataType)
				if err != nil {
					log.Warn().AnErr(fmt.Sprintf("Error converting bytes to string in matrix operation for feature %s", col), err)
					val = ""
				}
			} else {
				stringCol, ok := m.StringColumnIndexMap[col]
				if ok {
					val = row.StringData[stringCol.Index]
				} else {
					val = ""
				}
			}
			newRow[j] = val
		}
		matrix[i+1] = newRow
	}
	return matrix
}
