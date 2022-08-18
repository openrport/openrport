package symbols

import (
	"plugin"
	"sync"
)

// PluginSymbolTable maintains a thread safe list of plugin fns (aka symbols)
// loaded from the plugin for this capability.
type PluginSymbolTable struct {
	symbols map[string]plugin.Symbol

	sync.RWMutex
}

// NewSymbolTable creates a new symbol table of the specified size
func NewSymbolTable(tableSize int) (sym *PluginSymbolTable) {
	sym = &PluginSymbolTable{
		symbols: make(map[string]plugin.Symbol, tableSize),
	}
	return sym
}

// SetSymbol sets the plugin fn symbol for the name specified
func (sm *PluginSymbolTable) SetSymbol(name string, sym plugin.Symbol) {
	sm.Lock()
	defer sm.Unlock()
	sm.symbols[name] = sym
}

// GetSymbol gets the plugin fn symbol for the name specified
func (sm *PluginSymbolTable) GetSymbol(name string) (sym plugin.Symbol) {
	sm.RLock()
	defer sm.RUnlock()
	return sm.symbols[name]
}

// GetSymbolCount gets a count of the symbols loaded. Mainly used for
// testing purposes.
func (sm *PluginSymbolTable) GetSymbolCount() (count int) {
	sm.RLock()
	defer sm.RUnlock()
	return len(sm.symbols)
}
