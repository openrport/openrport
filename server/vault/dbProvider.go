package vault

import "sync"

type StatefulDbProviderFactory struct {
	initDBBuilder     func() (DbProvider, error)
	notInitDBProvider DbProvider
	initDBProvider    DbProvider
	providerLock      sync.RWMutex
}

func NewStatefulDbProviderFactory(initDBBuilder func() (DbProvider, error), notInitDBProvider DbProvider) DbProviderFactory {
	return &StatefulDbProviderFactory{
		initDBBuilder:     initDBBuilder,
		notInitDBProvider: notInitDBProvider,
		initDBProvider:    nil,
		providerLock:      sync.RWMutex{},
	}
}

func (dpf *StatefulDbProviderFactory) GetDbProvider() DbProvider {
	dpf.providerLock.RLock()
	defer dpf.providerLock.RUnlock()

	if dpf.initDBProvider != nil {
		return dpf.initDBProvider
	}

	return dpf.notInitDBProvider
}

func (dpf *StatefulDbProviderFactory) Init() error {
	dpf.providerLock.Lock()
	defer dpf.providerLock.Unlock()

	initDBProvider, err := dpf.initDBBuilder()
	if err != nil {
		return err
	}

	dpf.initDBProvider = initDBProvider

	return nil
}
