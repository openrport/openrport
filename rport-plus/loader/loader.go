package loader

import (
	"plugin"
)

func OpenPlugin(path string) (pin *plugin.Plugin, err error) {
	pin, err = plugin.Open(path)
	if err != nil {
		return nil, err
	}

	return pin, nil
}

func LoadSymbol(pluginPath string, name string) (sym plugin.Symbol, err error) {
	pin, err := OpenPlugin(pluginPath)
	if err != nil {
		return nil, err
	}

	sym, err = pin.Lookup(name)
	if err != nil {
		return nil, err
	}

	return sym, nil
}
