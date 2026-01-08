package random

import (
	"fmt"
	"math/rand"
	"strconv"
)

func GenerateRandomStringSlice(length int) []string {
	values := make([]string, length)
	for i := 0; i < length; i++ {
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, 3)
		for j := range b {
			b[j] = charset[rand.Intn(len(charset))]
		}
		values[i] = string(b)
	}
	return values
}
func GenerateRandomFloat32Slice(length int) []string {
	values := make([]string, length)
	for i := 0; i < length; i++ {
		// Use 'f' format to avoid scientific notation, with 6 decimal places
		values[i] = strconv.FormatFloat(float64(rand.Float32()), 'f', 7, 32)
	}
	return values
}
func GenerateRandomFloat64Slice(length int) []string {
	values := make([]string, length)
	for i := 0; i < length; i++ {
		// Use 'f' format to avoid scientific notation, with 6 decimal places
		values[i] = strconv.FormatFloat(rand.Float64(), 'f', 15, 64)
	}
	return values
}

func GenerateRandomInt32Slice(length int) []int {
	values := make([]int, length)
	for i := 0; i < length; i++ {
		values[i] = int(rand.Int31n(100))
	}
	return values
}

func GenerateRandomInt64Slice(length int) []int {
	values := make([]int, length)
	for i := 0; i < length; i++ {
		values[i] = int(rand.Int63n(100))
	}
	return values
}

func GenerateRandomUint32Slice(length int) []uint32 {
	values := make([]uint32, length)
	for i := 0; i < length; i++ {
		values[i] = uint32(rand.Uint32())
	}
	return values
}

func GenerateRandomUint64Slice(length int) []uint64 {
	values := make([]uint64, length)
	for i := 0; i < length; i++ {
		values[i] = uint64(rand.Uint64())
	}
	return values
}

func GenerateRandomBoolSlice(length int) []bool {
	values := make([]bool, length)
	for i := 0; i < length; i++ {
		values[i] = rand.Intn(2) == 1
	}
	return values
}

func generateRandomStringMatrixRecursive(dimensions []int, currentDim int) any {
	if currentDim == len(dimensions)-1 {
		values := make([]string, dimensions[currentDim])
		for i := 0; i < dimensions[currentDim]; i++ {
			const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			b := make([]byte, 3)
			for j := range b {
				b[j] = charset[rand.Intn(len(charset))]
			}
			values[i] = string(b)
		}
		return values
	}
	values := make([]any, dimensions[currentDim])
	for i := 0; i < dimensions[currentDim]; i++ {
		values[i] = generateRandomStringMatrixRecursive(dimensions, currentDim+1)
	}
	return values
}

func generateRandomFloat32MatrixRecursive(dimensions []int, currentDim int) any {
	if currentDim == len(dimensions)-1 {
		values := make([]float32, dimensions[currentDim])
		for i := 0; i < dimensions[currentDim]; i++ {
			values[i] = rand.Float32()
		}
		return values
	}
	values := make([]any, dimensions[currentDim])
	for i := 0; i < dimensions[currentDim]; i++ {
		values[i] = generateRandomFloat32MatrixRecursive(dimensions, currentDim+1)
	}
	return values
}

func generateRandomFloat64MatrixRecursive(dimensions []int, currentDim int) any {
	if currentDim == len(dimensions)-1 {
		values := make([]float64, dimensions[currentDim])
		for i := 0; i < dimensions[currentDim]; i++ {
			values[i] = rand.Float64()
		}
		return values
	}
	values := make([]any, dimensions[currentDim])
	for i := 0; i < dimensions[currentDim]; i++ {
		values[i] = generateRandomFloat64MatrixRecursive(dimensions, currentDim+1)
	}
	return values
}

func generateRandomInt32MatrixRecursive(dimensions []int, currentDim int) any {
	if currentDim == len(dimensions)-1 {
		values := make([]int32, dimensions[currentDim])
		for i := 0; i < dimensions[currentDim]; i++ {
			values[i] = rand.Int31n(100)
		}
		return values
	}
	values := make([]any, dimensions[currentDim])
	for i := 0; i < dimensions[currentDim]; i++ {
		values[i] = generateRandomInt32MatrixRecursive(dimensions, currentDim+1)
	}
	return values
}

func generateRandomInt64MatrixRecursive(dimensions []int, currentDim int) any {
	if currentDim == len(dimensions)-1 {
		values := make([]int64, dimensions[currentDim])
		for i := 0; i < dimensions[currentDim]; i++ {
			values[i] = rand.Int63n(100)
		}
		return values
	}
	values := make([]any, dimensions[currentDim])
	for i := 0; i < dimensions[currentDim]; i++ {
		values[i] = generateRandomInt64MatrixRecursive(dimensions, currentDim+1)
	}
	return values
}

func generateRandomUint32MatrixRecursive(dimensions []int, currentDim int) any {
	if currentDim == len(dimensions)-1 {
		values := make([]uint32, dimensions[currentDim])
		for i := 0; i < dimensions[currentDim]; i++ {
			values[i] = rand.Uint32()
		}
		return values
	}
	values := make([]any, dimensions[currentDim])
	for i := 0; i < dimensions[currentDim]; i++ {
		values[i] = generateRandomUint32MatrixRecursive(dimensions, currentDim+1)
	}
	return values
}

func generateRandomUint64MatrixRecursive(dimensions []int, currentDim int) any {
	if currentDim == len(dimensions)-1 {
		values := make([]uint64, dimensions[currentDim])
		for i := 0; i < dimensions[currentDim]; i++ {
			values[i] = rand.Uint64()
		}
		return values
	}
	values := make([]any, dimensions[currentDim])
	for i := 0; i < dimensions[currentDim]; i++ {
		values[i] = generateRandomUint64MatrixRecursive(dimensions, currentDim+1)
	}
	return values
}

func generateRandomBoolMatrixRecursive(dimensions []int, currentDim int) any {
	if currentDim == len(dimensions)-1 {
		values := make([]bool, dimensions[currentDim])
		for i := 0; i < dimensions[currentDim]; i++ {
			values[i] = rand.Intn(2) == 1
		}
		return values
	}
	values := make([]any, dimensions[currentDim])
	for i := 0; i < dimensions[currentDim]; i++ {
		values[i] = generateRandomBoolMatrixRecursive(dimensions, currentDim+1)
	}
	return values
}

func generateRandomBytesMatrixRecursive(dimensions []int, currentDim int) any {

	if currentDim == len(dimensions)-1 {
		values := make([]string, dimensions[currentDim])
		for i := 0; i < dimensions[currentDim]; i++ {
			values[i] = strconv.FormatFloat(rand.Float64(), 'f', -1, 32)
		}
		return values
	}
	values := make([]any, dimensions[currentDim])
	for i := 0; i < dimensions[currentDim]; i++ {
		values[i] = generateRandomBytesMatrixRecursive(dimensions, currentDim+1)
	}
	return values
}

func GenerateRandomStringMatrix(dimensions []int) any {
	if len(dimensions) == 0 {
		return nil
	}
	return generateRandomStringMatrixRecursive(dimensions, 0)
}

func GenerateRandomFloat32Matrix(dimensions []int) any {
	if len(dimensions) == 0 {
		return nil
	}
	return generateRandomFloat32MatrixRecursive(dimensions, 0)
}

func GenerateRandomFloat64Matrix(dimensions []int) any {
	if len(dimensions) == 0 {
		return nil
	}
	return generateRandomFloat64MatrixRecursive(dimensions, 0)
}

func GenerateRandomInt32Matrix(dimensions []int) any {
	if len(dimensions) == 0 {
		return nil
	}
	return generateRandomInt32MatrixRecursive(dimensions, 0)
}

func GenerateRandomInt64Matrix(dimensions []int) any {
	if len(dimensions) == 0 {
		return nil
	}
	return generateRandomInt64MatrixRecursive(dimensions, 0)
}

func GenerateRandomUint32Matrix(dimensions []int) any {
	if len(dimensions) == 0 {
		return nil
	}
	return generateRandomUint32MatrixRecursive(dimensions, 0)
}

func GenerateRandomUint64Matrix(dimensions []int) any {
	if len(dimensions) == 0 {
		return nil
	}
	return generateRandomUint64MatrixRecursive(dimensions, 0)
}

func GenerateRandomBoolMatrix(dimensions []int) any {
	if len(dimensions) == 0 {
		return nil
	}
	return generateRandomBoolMatrixRecursive(dimensions, 0)
}

func GenerateRandomBytesMatrix(dimensions []int) any {
	if len(dimensions) == 0 {
		return nil
	}
	return generateRandomBytesMatrixRecursive(dimensions, 0)
}

func GenerateRandom(dimensions []int, dataType string) any {
	switch dataType {
	case "FP32", "FP16", "FP8":
		return GenerateRandomFloat32Matrix(dimensions)
	case "FP64":
		return GenerateRandomFloat64Matrix(dimensions)
	case "INT32", "INT16", "INT8":
		return GenerateRandomInt32Matrix(dimensions)
	case "INT64":
		return GenerateRandomInt64Matrix(dimensions)
	case "UINT32", "UINT16", "UINT8":
		return GenerateRandomUint32Matrix(dimensions)
	case "UINT64":
		return GenerateRandomUint64Matrix(dimensions)
	case "BOOL":
		return GenerateRandomBoolMatrix(dimensions)
	case "STRING":
		return GenerateRandomStringMatrix(dimensions)
	case "BYTES":
		return GenerateRandomBytesMatrix(dimensions)
	default:
		return nil
	}
}

func GenerateRandomInteWithRange(min int, max int) string {
	value := rand.Intn(max-min+1) + min
	return fmt.Sprintf("%d", value)
}

func GenerateRandomIntSliceWithRange(length int, min int, max int) []string {
	values := make([]string, length)
	for i := 0; i < length; i++ {
		values[i] = fmt.Sprintf("%d", rand.Intn(max-min+1)+min)
	}
	return values
}
