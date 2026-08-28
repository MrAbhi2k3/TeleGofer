package crypto

import (
	"math/big"
)

// FactorPQ factors a 64-bit semiprime pq into primes p and q such that p < q.
// Uses Pollard's rho algorithm with Floyd's cycle-finding method.
func FactorPQ(pq uint64) (uint64, uint64) {
	if pq%2 == 0 {
		return 2, pq / 2
	}

	var (
		x   uint64 = 2
		y   uint64 = 2
		c   uint64 = 1
		d   uint64 = 1
		one        = big.NewInt(1)
	)

	f := func(val, n uint64) uint64 {
		// (val*val + c) % n
		v := new(big.Int).SetUint64(val)
		v.Mul(v, v)
		v.Add(v, new(big.Int).SetUint64(c))
		v.Mod(v, new(big.Int).SetUint64(n))
		return v.Uint64()
	}

	gcd := func(a, b uint64) uint64 {
		g := new(big.Int).GCD(nil, nil, new(big.Int).SetUint64(a), new(big.Int).SetUint64(b))
		return g.Uint64()
	}

	for d == 1 {
		x = f(x, pq)
		y = f(f(y, pq), pq)

		var diff uint64
		if x > y {
			diff = x - y
		} else {
			diff = y - x
		}

		d = gcd(diff, pq)

		if d == pq {
			// Retry with different polynomial constant c
			c++
			x = 2
			y = 2
			d = 1
		}
	}

	p := d
	q := pq / d
	if p > q {
		p, q = q, p
	}
	_ = one

	return p, q
}
