package util

import (
	"bytes"
	"sync"
)

var BufferPool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

var BoolSlicePool = &sync.Pool{
	New: func() interface{} {
		return []bool{}
	},
}
