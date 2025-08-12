//go:build amd64 && avx2
// +build amd64,avx2

#include "textflag.h"

// func match16_simd(ctrl *byte, h2 byte) uint16
TEXT ·match16_simd(SB),NOSPLIT,$0-0
    // DI = &ctrl[0];    SIL = h2 (byte parameter)

    // Load 16 control bytes into Y0
    VMOVDQU (DI), Y0

    // Broadcast h2 from memory operand directly into Y1
    VPBROADCASTB h2+8(FP), Y1

    // Compare Y0 bytes with broadcasted h2
    VPCMPEQB Y1, Y0, Y2

    // Extract the MSBs of comparison result into AX as 16‑bit mask
    VPMOVMSKB Y2, AX

    VZEROUPPER
    RET
