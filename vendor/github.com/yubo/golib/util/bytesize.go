package util

import (
	"fmt"
)

type ByteSize uint64

const (
	B  ByteSize = 1
	KB          = B << 10
	MB          = KB << 10
	GB          = MB << 10
	TB          = GB << 10
	PB          = TB << 10
	EB          = PB << 10

	fnUnmarshalText string = "UnmarshalText"
	maxUint64       uint64 = (1 << 64) - 1
	cutoff          uint64 = maxUint64 / 10
)

func (b ByteSize) Bytes() uint64 {
	return uint64(b)
}

func (b ByteSize) KBytes() float64 {
	v := b / KB
	r := b % KB
	return float64(v) + float64(r)/float64(KB)
}

func (b ByteSize) MBytes() float64 {
	v := b / MB
	r := b % MB
	return float64(v) + float64(r)/float64(MB)
}

func (b ByteSize) GBytes() float64 {
	v := b / GB
	r := b % GB
	return float64(v) + float64(r)/float64(GB)
}

func (b ByteSize) TBytes() float64 {
	v := b / TB
	r := b % TB
	return float64(v) + float64(r)/float64(TB)
}

func (b ByteSize) PBytes() float64 {
	v := b / PB
	r := b % PB
	return float64(v) + float64(r)/float64(PB)
}

func (b ByteSize) EBytes() float64 {
	v := b / EB
	r := b % EB
	return float64(v) + float64(r)/float64(EB)
}

func (b ByteSize) String() string {
	switch {
	case b == 0:
		return fmt.Sprint("0B")
	case b%EB == 0:
		return fmt.Sprintf("%dEB", b/EB)
	case b%PB == 0:
		return fmt.Sprintf("%dPB", b/PB)
	case b%TB == 0:
		return fmt.Sprintf("%dTB", b/TB)
	case b%GB == 0:
		return fmt.Sprintf("%dGB", b/GB)
	case b%MB == 0:
		return fmt.Sprintf("%dMB", b/MB)
	case b%KB == 0:
		return fmt.Sprintf("%dKB", b/KB)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func (b ByteSize) HR() string {
	return b.HumanReadable()
}

func (b ByteSize) HumanReadable() string {
	switch {
	case b > EB:
		return fmt.Sprintf("%.1f EB", b.EBytes())
	case b > PB:
		return fmt.Sprintf("%.1f PB", b.PBytes())
	case b > TB:
		return fmt.Sprintf("%.1f TB", b.TBytes())
	case b > GB:
		return fmt.Sprintf("%.1f GB", b.GBytes())
	case b > MB:
		return fmt.Sprintf("%.1f MB", b.MBytes())
	case b > KB:
		return fmt.Sprintf("%.1f KB", b.KBytes())
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func SizeOf(v string) uint64 {
	n := uint64(toInt64(v[:len(v)-1]))

	switch byte(v[len(v)-1]) {
	case 'k', 'K':
		n *= uint64(KB)
	case 'm', 'M':
		n *= uint64(MB)
	case 'g', 'G':
		n *= uint64(GB)
	case 't', 'T':
		n *= uint64(TB)
	case 'p', 'P':
		n *= uint64(PB)
	case 'e', 'E':
		n *= uint64(EB)
	default:
		n = uint64(toInt64(v))
	}
	return n
}

func SizeReadable(b int) string {
	return ByteSize(b).HumanReadable()
}
