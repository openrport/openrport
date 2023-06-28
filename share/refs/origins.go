package refs

import (
	"fmt"
	"strings"
)

const (
	Head      = "Origin(Source("
	Separator = "), Parent("
	Foot      = "))"
)

type Origin interface {
	Source() identifiable
	Parent() identifiable
	NextFromIdentifiable(nextParent identifiable) origin
	Next(iType IdentifiableType, id string) origin
	String() string
}

type origin struct {
	source identifiable
	parent identifiable
}

func (o origin) Source() identifiable {
	return o.source
}

func (o origin) Parent() identifiable {
	return o.parent
}

//nolint:revive
func (o origin) NextFromIdentifiable(nextParent identifiable) origin {
	return origin{
		source: o.source,
		parent: nextParent,
	}
}

//nolint:revive
func (o origin) Next(iType IdentifiableType, id string) origin {
	return origin{source: o.source, parent: NewIdentifiable(iType, id)}
}

func (o origin) String() string {
	return Head + o.source.String() + Separator + o.parent.String() + Foot
}

// nolint:revive
func NewOrigin(source identifiable, parent identifiable) origin {
	return origin{source: source, parent: parent}
}

//nolint:revive
func ParseOrigin(raw string) (origin, error) {

	if len(raw) < len(Head)+len(Foot) {
		return origin{}, fmt.Errorf("can't parse origin: %v", raw)
	}
	inside := raw[len(Head) : len(raw)-len(Foot)]
	parts := strings.Split(inside, Separator)
	if len(parts) != 2 {
		return origin{}, fmt.Errorf("can't parse origin: %v", raw)
	}
	source, err := ParseIdentifiable(parts[0])
	if err != nil {
		return origin{}, fmt.Errorf("can't parse origin: %v", err)
	}
	parent, err := ParseIdentifiable(parts[1])
	if err != nil {
		return origin{}, fmt.Errorf("can't parse origin: %v", err)
	}

	return origin{source: source, parent: parent}, nil
}
