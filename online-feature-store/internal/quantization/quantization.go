package quantization

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
)

// quantize processes a 3D byte matrix and converts its elements based on the specified source and requested data types for each column.
//
// Parameters:
// - matrix: 3D matrix containing all the feature values
// - columnSourceDatatypeMap: A map where the key is the column name and the value is the source data type for that column.
// - columnRequestedDatatypeMap: A map where the key is the column name and the value is the requested data type for that column.
// - matrixColumnIndexMap: A map where the key is the column name and the value is the index of that column in the matrix.
func quantize(matrix [][][]byte,
	columnSourceDatatypeMap map[string]types.DataType,
	columnRequestedDatatypeMap map[string]types.DataType, matrixColumnIndexMap map[string]int) {

	rows := len(matrix)

	for columnName, requiredDataType := range columnRequestedDatatypeMap {

		columnIndex := matrixColumnIndexMap[columnName]

		quantizeByteSlice, _ := GetQuantizationFunction(columnSourceDatatypeMap[columnName], requiredDataType)

		for i := 0; i < rows; i++ {
			matrix[i][columnIndex] = quantizeByteSlice(matrix[i][columnIndex])
		}
	}

}
