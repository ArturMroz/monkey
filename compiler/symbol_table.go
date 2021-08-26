package compiler

type SymbolScope string

const (
	GlobalScope  SymbolScope = "GLOBAL"
	LocalScope   SymbolScope = "LOCAL"
	BuiltinScope SymbolScope = "BUILTIN"
	FreeScope    SymbolScope = "FREE"
)

type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}

type SymbolTable struct {
	Outer          *SymbolTable
	FreeSymbols    []Symbol
	store          map[string]Symbol
	numDefinitions int
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		store:       map[string]Symbol{},
		FreeSymbols: []Symbol{},
	}
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Outer: outer,
		store: map[string]Symbol{},
	}
}

func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{Name: name, Index: s.numDefinitions}

	if s.Outer == nil {
		symbol.Scope = GlobalScope
	} else {
		symbol.Scope = LocalScope
	}

	s.store[name] = symbol
	s.numDefinitions++

	return symbol
}

func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{Name: name, Index: index, Scope: BuiltinScope}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.Resolve(name)
		if !ok {
			return obj, false
		}

		if obj.Scope == GlobalScope || obj.Scope == BuiltinScope {
			return obj, true
		}

		free := s.defineFree(obj)
		return free, true
	}

	return obj, ok
}

func (s *SymbolTable) defineFree(original Symbol) Symbol {
	// TODO could just add free symbol instead of this replace dance?
	s.FreeSymbols = append(s.FreeSymbols, original)

	symbol := Symbol{
		Name:  original.Name,
		Index: len(s.FreeSymbols) - 1,
		Scope: FreeScope,
	}

	s.store[original.Name] = symbol
	return symbol
}
