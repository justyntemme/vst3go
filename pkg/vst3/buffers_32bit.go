//go:build 386 || arm || mips || mipsle

package vst3

const MaxArraySize = 1 << 26 // 67,108,864 elements for 32-bit systems