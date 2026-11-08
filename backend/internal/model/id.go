package model

import (
	"crypto/rand"
	"math/big"
	"time"
)

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewID 生成 26 位 ULID（48 位毫秒时间戳 + 80 位随机后缀），用于全部主键。
// 时间有序便于排序与排查，无需额外依赖。
func NewID() string {
	ts := uint64(time.Now().UTC().UnixMilli())
	var entropy [10]byte
	_, _ = rand.Read(entropy[:])

	out := make([]byte, 26)
	for i := 9; i >= 0; i-- {
		out[i] = crockford[ts&31]
		ts >>= 5
	}

	n := new(big.Int).SetBytes(entropy[:])
	mask := big.NewInt(31)
	for i := 25; i >= 10; i-- {
		out[i] = crockford[new(big.Int).And(n, mask).Uint64()]
		n.Rsh(n, 5)
	}
	return string(out)
}
