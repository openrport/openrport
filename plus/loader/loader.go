package loader

import (
	"plugin"
)

// LoadSymbol opens the plugin and looks up the symbol with the name specified.
func LoadSymbol(pluginPath string, name string) (sym plugin.Symbol, err error) {
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
