package refs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
)

// IdentifiableType make it easier for ide to find defined instances in the codebase, type can't contain (
type IdentifiableType string

type Identifiable interface {
	// Type know what it is
	Type() IdentifiableType
	// ID something by which you can find it in it's origin storage, can't contain )
	ID() string
	// String make it serializable nicely for others to store in DB and for you to reconstruct later
	String() string
}

//nolint:revive
func ParseIdentifiable(raw string) (identifiable, error) {
	parts := strings.Split(raw, ":::")

	if len(parts) != 2 {
		return identifiable{}, fmt.Errorf("cant parse identifielbe: %v", raw)
	}

	return identifiable{
		iType: IdentifiableType(parts[0]),
		id:    parts[1],
	}, nil
}

type identifiable struct {
	iType IdentifiableType
	id    string
}

//nolint:revive
func GenerateIdentifiable(iType IdentifiableType) identifiable {
	id := ulid.Make()
	return NewIdentifiable(iType, id.String())
}

//nolint:revive
func NewIdentifiable(iType IdentifiableType, id string) identifiable {
	return identifiable{iType: iType, id: id}
}

// Type know what it is
func (i identifiable) Type() IdentifiableType {
	return i.iType
}

// ID something by which you can find it in its origin storage
func (i identifiable) ID() string {
	return i.id
}

// String make it serializable nicely for others to store in DB and for you to reconstruct later
func (i identifiable) String() string {
	return string(i.iType) + ":::" + i.id
}

// MarshalJSON allows identifiable to be serialized to json
func (i identifiable) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

//nolint:revive
func MustIdentifiableFactory(iType IdentifiableType) func(id string) identifiable {
	return func(id string) identifiable {
		return NewIdentifiable(iType, id)
	}
}

//nolint:revive
func MustGenerator(iType IdentifiableType) func() identifiable {
	return func() identifiable {
		return GenerateIdentifiable(iType)
	}
}
