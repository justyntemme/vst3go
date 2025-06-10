//go:build amd64 || arm64 || ppc64 || ppc64le || mips64 || mips64le || s390x || sparc64 || riscv64

package vst3

const MaxArraySize = 1 << 30 // 1,073,741,824 elements for 64-bit systems