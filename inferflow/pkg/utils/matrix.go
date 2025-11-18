package utils

//
//import (
//	"fmt"
//	"math/rand"
//
//	"github.com/emirpasic/gods/maps/linkedhashmap"
//)
//
//type ComponentMatrix struct {
//	Matrix               [][]string
//	matrixColumnIndexMap map[string]int
//}
//
//func (m *ComponentMatrix) InitMatrix(rowcount int, colcount int, columnnames []string) [][]string {
//
//	m.Matrix = make([][]string, rowcount+1)
//	for i := range m.Matrix {
//		m.Matrix[i] = make([]string, colcount)
//	}
//
//	m.matrixColumnIndexMap = make(map[string]int)
//	for i := 0; i < colcount; i++ {
//		m.Matrix[0][i] = columnnames[i]
//		m.matrixColumnIndexMap[columnnames[i]] = i
//	}
//	return m.Matrix
//}
//
//func (m *ComponentMatrix) Get(row, column int) string {
//	return m.Matrix[row][column]
//}
//
//func (m *ComponentMatrix) Set(row, column int, value string) {
//	m.Matrix[row][column] = value
//}
//
//func (m *ComponentMatrix) PopulateByteDataFromMap(columnToIterate string, columnToPopulate string, valuemap map[string]string) {
//
//	if columnToIterateIndex, ok := m.matrixColumnIndexMap[columnToIterate]; ok {
//		if columnToPopulateIndex, ok := m.matrixColumnIndexMap[columnToPopulate]; ok {
//			for i := 1; i < len(m.Matrix); i++ {
//				key := m.Get(i, columnToIterateIndex)
//				m.Set(i, columnToPopulateIndex, valuemap[key])
//			}
//		}
//	}
//}
//
//func (m *ComponentMatrix) PopulateColumn(columnToPopulate string, values []string) {
//
//	if columnToPopulateIndex, ok := m.matrixColumnIndexMap[columnToPopulate]; ok {
//		for i := 1; i < len(m.Matrix); i++ {
//			m.Set(i, columnToPopulateIndex, values[i-1])
//		}
//	}
//}
//
//func (m *ComponentMatrix) PopulateColumnFromSingleValue(columnToPopulate string, value string) {
//
//	if columnToPopulateIndex, ok := m.matrixColumnIndexMap[columnToPopulate]; ok {
//		for i := 1; i < len(m.Matrix); i++ {
//			m.Set(i, columnToPopulateIndex, value)
//		}
//	}
//}
//
///*
// * method randomly selects one column from the sourceColumns
// * and populates another column (columnToPopulate) in the ComponentMatrix object (m).
// */
//func (m *ComponentMatrix) PopulateColRandomlyFromSourceCols(columnToIterate string, columnToPopulate string, sourceColumns []string) {
//
//	if columnToPopulateIndex, ok := m.matrixColumnIndexMap[columnToPopulate]; ok {
//		for i := 1; i < len(m.Matrix); i++ {
//			randomIndex := rand.Intn(len(sourceColumns))
//			if columnToIterateIndex, ok := m.matrixColumnIndexMap[sourceColumns[randomIndex]]; ok {
//				value := m.Get(i, columnToIterateIndex)
//				m.Set(i, columnToPopulateIndex, value)
//			}
//		}
//	}
//}
//
///*
// * method populates another column (columnToPopulate) in the ComponentMatrix object (m) using below logic
// * if source col 1 and source col 2 are non zero then populate new col with source col 1
// * otherwise populate new col with source col 2
// */
//func (m *ComponentMatrix) PopulateColFromSourceCols(columnToIterate string, columnToPopulate string, sourceColumns []string) {
//
//	if columnToPopulateIndex, ok := m.matrixColumnIndexMap[columnToPopulate]; ok {
//		for i := 1; i < len(m.Matrix); i++ {
//			var value1 string
//			var value2 string
//			if columnToIterateIndex, ok := m.matrixColumnIndexMap[sourceColumns[0]]; ok {
//				value1 = m.Get(i, columnToIterateIndex)
//			}
//			if columnToIterateIndex, ok := m.matrixColumnIndexMap[sourceColumns[1]]; ok {
//				value2 = m.Get(i, columnToIterateIndex)
//			}
//			if !IsNilOrEmpty(value1) && !IsNilOrEmpty(value2) && !IsZeroValue(value1) && !IsZeroValue(value2) {
//				m.Set(i, columnToPopulateIndex, value1)
//			} else {
//				m.Set(i, columnToPopulateIndex, value2)
//			}
//		}
//	}
//}
//
//func (m *ComponentMatrix) GetColumnValues(columns []string) [][]string {
//
//	results := make([][]string, len(columns))
//	for i, column := range columns {
//		result := make([]string, 0)
//		if columnIndex, ok := m.matrixColumnIndexMap[column]; ok {
//			for j := 1; j < len(m.Matrix); j++ {
//				value := m.Get(j, columnIndex)
//				result = append(result, value)
//			}
//		}
//		results[i] = result
//	}
//	return results
//}
//
//func (m *ComponentMatrix) GetColumnValuesWithKey(columnsWithKey *linkedhashmap.Map) []map[interface{}][]string {
//
//	results := make([]map[interface{}][]string, 0)
//	for _, key := range columnsWithKey.StringData() {
//		value, _ := columnsWithKey.Get(key)
//		colValues := m.GetColumnValues([]string{value.(string)})[0]
//		result := make(map[interface{}][]string)
//		result[key] = colValues
//		results = append(results, result)
//	}
//	return results
//}
//
//func (m *ComponentMatrix) GetMatrixOfColumnSlice(columns []string, defaultVal bool) [][]string {
//
//	// create a new 2D matrix with the selected columns
//	newMatrix := make([][]string, len(m.Matrix))
//	for i, row := range m.Matrix {
//		newRow := make([]string, len(columns))
//		for j, column := range columns {
//			if index, ok := m.matrixColumnIndexMap[column]; ok {
//				// default value in case of FS call failure
//				// TODO feature wise default fetch from config
//				if defaultVal && row[index] == "" {
//					newRow[j] = "0.0"
//				} else {
//					newRow[j] = row[index]
//				}
//			}
//		}
//		newMatrix[i] = newRow
//	}
//	return newMatrix
//}
//
//func (m *ComponentMatrix) Print() {
//
//	for i := 0; i < len(m.Matrix); i++ {
//		for j := 0; j < len(m.Matrix[0]); j++ {
//			fmt.Printf("%4s  ", m.Matrix[i][j])
//		}
//		fmt.Println()
//	}
//}
