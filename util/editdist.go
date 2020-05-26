package util

import (
	"github.com/piggy-moe/subtitle/config"
)

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func EditDistance(ss, tt []rune) int {
	m := len(ss) + 1
	n := len(tt) + 1
	d := make([][]int, m)
	for i := range d {
		d[i] = make([]int, n)
	}
	for i := 0; i < m; i++ {
		d[i][0] = i
	}
	for j := 0; j < n; j++ {
		d[0][j] = j
	}
	for j := 1; j < n; j++ {
		for i := 1; i < m; i++ {
			var c int
			if ss[i-1] == tt[j-1] {
				c = 0
			} else {
				c = 1
			}
			d[i][j] = min(d[i-1][j-1]+c, min(d[i-1][j]+1, d[i][j-1]+1))
		}
	}
	return d[m-1][n-1]
}

func Silimar(s, t string, rv config.RelativeValue) bool {
	if rv.Equal(0, 0) {
		return s == t
	}
	ss := []rune(s)
	tt := []rune(t)
	return EditDistance(ss, tt) <= rv.Calculate(len(ss))
}
