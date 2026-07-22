package geosenv

import (
	"sync"

	"github.com/twpayne/go-geos"
)

var contextPool = sync.Pool{
	New: func() interface{} {
		return geos.NewContext()
	},
}

func GetContext() *geos.Context {
	return contextPool.Get().(*geos.Context)
}

func ReleaseContext(ctx *geos.Context) {
	contextPool.Put(ctx)
}
