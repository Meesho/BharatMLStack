package float8

import (
	"math"
	"testing"
)

func TestAllFP8E4M3ToFP32Value(t *testing.T) {

	// Test with all possible FP8 values [uint8: 0 to 255]

	tests := []struct {
		val      Float8e4m3
		expected float32
	}{
		{0, 0}, {1, 0.001953125}, {2, 0.00390625}, {3, 0.005859375}, {4, 0.0078125}, {5, 0.009765625}, {6, 0.01171875}, {7, 0.013671875}, {8, 0.015625}, {9, 0.017578125}, {10, 0.01953125}, {11, 0.021484375}, {12, 0.0234375}, {13, 0.025390625}, {14, 0.02734375}, {15, 0.029296875}, {16, 0.03125}, {17, 0.03515625}, {18, 0.0390625}, {19, 0.04296875}, {20, 0.046875}, {21, 0.05078125}, {22, 0.0546875}, {23, 0.05859375}, {24, 0.0625}, {25, 0.0703125}, {26, 0.078125}, {27, 0.0859375}, {28, 0.09375}, {29, 0.1015625}, {30, 0.109375}, {31, 0.1171875}, {32, 0.125}, {33, 0.140625}, {34, 0.15625}, {35, 0.171875}, {36, 0.1875}, {37, 0.203125}, {38, 0.21875}, {39, 0.234375}, {40, 0.25}, {41, 0.28125}, {42, 0.3125}, {43, 0.34375}, {44, 0.375}, {45, 0.40625}, {46, 0.4375}, {47, 0.46875}, {48, 0.5}, {49, 0.5625}, {50, 0.625}, {51, 0.6875}, {52, 0.75}, {53, 0.8125}, {54, 0.875}, {55, 0.9375}, {56, 1}, {57, 1.125}, {58, 1.25}, {59, 1.375}, {60, 1.5}, {61, 1.625}, {62, 1.75}, {63, 1.875}, {64, 2}, {65, 2.25}, {66, 2.5}, {67, 2.75}, {68, 3}, {69, 3.25}, {70, 3.5}, {71, 3.75}, {72, 4}, {73, 4.5}, {74, 5}, {75, 5.5}, {76, 6}, {77, 6.5}, {78, 7}, {79, 7.5}, {80, 8}, {81, 9}, {82, 10}, {83, 11}, {84, 12}, {85, 13}, {86, 14}, {87, 15}, {88, 16}, {89, 18}, {90, 20}, {91, 22}, {92, 24}, {93, 26}, {94, 28}, {95, 30}, {96, 32}, {97, 36}, {98, 40}, {99, 44}, {100, 48}, {101, 52}, {102, 56}, {103, 60}, {104, 64}, {105, 72}, {106, 80}, {107, 88}, {108, 96}, {109, 104}, {110, 112}, {111, 120}, {112, 128}, {113, 144}, {114, 160}, {115, 176}, {116, 192}, {117, 208}, {118, 224}, {119, 240}, {120, 256}, {121, 288}, {122, 320}, {123, 352}, {124, 384}, {125, 416}, {126, 448}, {127, float32(math.NaN())}, {128, -0}, {129, -0.001953125}, {130, -0.00390625}, {131, -0.005859375}, {132, -0.0078125}, {133, -0.009765625}, {134, -0.01171875}, {135, -0.013671875}, {136, -0.015625}, {137, -0.017578125}, {138, -0.01953125}, {139, -0.021484375}, {140, -0.0234375}, {141, -0.025390625}, {142, -0.02734375}, {143, -0.029296875}, {144, -0.03125}, {145, -0.03515625}, {146, -0.0390625}, {147, -0.04296875}, {148, -0.046875}, {149, -0.05078125}, {150, -0.0546875}, {151, -0.05859375}, {152, -0.0625}, {153, -0.0703125}, {154, -0.078125}, {155, -0.0859375}, {156, -0.09375}, {157, -0.1015625}, {158, -0.109375}, {159, -0.1171875}, {160, -0.125}, {161, -0.140625}, {162, -0.15625}, {163, -0.171875}, {164, -0.1875}, {165, -0.203125}, {166, -0.21875}, {167, -0.234375}, {168, -0.25}, {169, -0.28125}, {170, -0.3125}, {171, -0.34375}, {172, -0.375}, {173, -0.40625}, {174, -0.4375}, {175, -0.46875}, {176, -0.5}, {177, -0.5625}, {178, -0.625}, {179, -0.6875}, {180, -0.75}, {181, -0.8125}, {182, -0.875}, {183, -0.9375}, {184, -1}, {185, -1.125}, {186, -1.25}, {187, -1.375}, {188, -1.5}, {189, -1.625}, {190, -1.75}, {191, -1.875}, {192, -2}, {193, -2.25}, {194, -2.5}, {195, -2.75}, {196, -3}, {197, -3.25}, {198, -3.5}, {199, -3.75}, {200, -4}, {201, -4.5}, {202, -5}, {203, -5.5}, {204, -6}, {205, -6.5}, {206, -7}, {207, -7.5}, {208, -8}, {209, -9}, {210, -10}, {211, -11}, {212, -12}, {213, -13}, {214, -14}, {215, -15}, {216, -16}, {217, -18}, {218, -20}, {219, -22}, {220, -24}, {221, -26}, {222, -28}, {223, -30}, {224, -32}, {225, -36}, {226, -40}, {227, -44}, {228, -48}, {229, -52}, {230, -56}, {231, -60}, {232, -64}, {233, -72}, {234, -80}, {235, -88}, {236, -96}, {237, -104}, {238, -112}, {239, -120}, {240, -128}, {241, -144}, {242, -160}, {243, -176}, {244, -192}, {245, -208}, {246, -224}, {247, -240}, {248, -256}, {249, -288}, {250, -320}, {251, -352}, {252, -384}, {253, -416}, {254, -448}, {255, float32(math.NaN())},
	}

	for _, test := range tests {

		result := FP8E4M3ToFP32Value(test.val)

		if !(math.IsNaN(float64(result)) && math.IsNaN(float64(test.expected)) || result == test.expected) {
			t.Errorf("FP8E4M3ToFP32Value failed. Expected %v, got %v", test.expected, result)
		}
	}

}

func TestFP8E4M3FromFP32Value(t *testing.T) {

	tests := []struct {
		val      float32
		expected Float8e4m3
	}{
		{0.0039537125, 2}, // random value within range
		{448, 126},        // max normal
		{0.015625, 8},     // min normal
		{5000, 127},       // overflow
		{0, 0},            // zero
		{0.013671875, 7},  // max sub-normal
		{0.001953125, 1},  // min sub-normal
		{0.0001953125, 0}, // underflow
	}

	for _, test := range tests {

		result := FP8E4M3FromFP32Value(test.val)

		if result != test.expected {
			t.Errorf("FP8E4M3FromFP32Value failed. Expected %v, got %v", test.expected, result)
		}
	}

}

// Following tests were generated by Copilot
func TestFP8E4M3FromFP32Value_PositiveInfinity(t *testing.T) {
	result := FP8E4M3FromFP32Value(float32(math.Inf(1)))
	if result != 0x7F {
		t.Errorf("Expected 0x7F, got %02X", result)
	}
}

func TestFP8E4M3FromFP32Value_NegativeInfinity(t *testing.T) {
	result := FP8E4M3FromFP32Value(float32(math.Inf(-1)))
	if result != 0xFF {
		t.Errorf("Expected 0xFF, got %02X", result)
	}
}

func TestFP8E4M3FromFP32Value_PositiveNormal(t *testing.T) {
	result := FP8E4M3FromFP32Value(1.0)
	if result != 0x38 {
		t.Errorf("Expected 0x38, got %02X", result)
	}
}

func TestFP8E4M3FromFP32Value_NegativeNormal(t *testing.T) {
	result := FP8E4M3FromFP32Value(-1.0)
	if result != 0xB8 {
		t.Errorf("Expected 0xB8, got %02X", result)
	}
}
