package loader

import (
	"plugin"
)

type Loader interface {
	LoadSymbol(pluginPath string, name string) (sym plugin.Symbol, err error)
}

type PluginLoader struct{}

func New() (pl *PluginLoader) {
	pl = &PluginLoader{}
	return pl
}

// LoadSymbol opens the plugin and looks up the symbol with the name specified.
func (pl *PluginLoader) LoadSymbol(pluginPath string, name string) (sym plugin.Symbol, err error) {
	pin, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, err
	}

	sym, err = pin.Lookup(name)
	if err != nil {
		return nil, err
	}

	return sym, nil
}
