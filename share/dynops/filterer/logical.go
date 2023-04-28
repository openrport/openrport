package filterer

type Operation[T any] interface {
	Run(T) bool
}

//////// False

type FalseOp[T any] struct {
}

func (f FalseOp[T]) Run(T) bool {
	return false
}

func NewFalse[T any]() FalseOp[T] {
	return FalseOp[T]{}
}

/////// True

type TrueOp[T any] struct {
}

func (f TrueOp[T]) Run(T) bool {
	return true
}

func NewTrue[T any]() TrueOp[T] {
	return TrueOp[T]{}
}

/////// Check bool

type BoolOp struct {
}

func (f BoolOp) Run(v bool) bool {
	return v
}

func NewBool() BoolOp {
	return BoolOp{}
}

////// AND

type And[T any] struct {
	ops []Operation[T]
}

func (a And[T]) Run(o T) bool {
	if len(a.ops) == 0 {
		return false
	}

	for _, op := range a.ops {
		if !op.Run(o) {
			return false
		}
	}

	return true
}

func NewAnd[T any](ops ...Operation[T]) And[T] {
	return And[T]{ops: ops}
}

////// OR

type Or[T any] struct {
	ops []Operation[T]
}

func (a Or[T]) Run(o T) bool {

	for _, op := range a.ops {
		if op.Run(o) {
			return true
		}
	}

	return false
}

func NewOr[T any](ops ...Operation[T]) Or[T] {
	return Or[T]{ops: ops}
}
