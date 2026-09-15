package durable

type LayerLocker interface {
	Lock(path string) (func() error, error)
}

type osLayerLocker struct{}

func (osLayerLocker) Lock(path string) (func() error, error) {
	return acquireLayerLock(path)
}

var DefaultLayerLocker LayerLocker = osLayerLocker{}
