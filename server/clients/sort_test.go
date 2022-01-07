package clients

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortByIDAsc(t *testing.T) {
	// given
	a := []*CalculatedClient{c1.ToCalculated(nil), c2.ToCalculated(nil), c3.ToCalculated(nil), c4.ToCalculated(nil)}

	// when
	SortByID(a, false)

	// then
	assert.ElementsMatch(t, a, []*CalculatedClient{c2.ToCalculated(nil), c4.ToCalculated(nil), c1.ToCalculated(nil), c3.ToCalculated(nil)})
}

func TestSortByIDDesc(t *testing.T) {
	// given
	a := []*CalculatedClient{c1.ToCalculated(nil), c2.ToCalculated(nil), c3.ToCalculated(nil), c4.ToCalculated(nil)}

	// when
	SortByID(a, true)

	// then
	assert.ElementsMatch(t, a, []*CalculatedClient{c3.ToCalculated(nil), c1.ToCalculated(nil), c4.ToCalculated(nil), c2.ToCalculated(nil)})
}

var (
	c1N = &CalculatedClient{Client: &Client{ID: "a1", Name: "name12"}}
	c2N = &CalculatedClient{Client: &Client{ID: "A2", Name: "Name12"}}
	c3N = &CalculatedClient{Client: &Client{ID: "a3", Name: "name34"}}
	c4N = &CalculatedClient{Client: &Client{ID: "a4", Name: "name34"}}
	c5N = &CalculatedClient{Client: &Client{ID: "A5", Name: "Name5"}}
	c6N = &CalculatedClient{Client: &Client{ID: "a6", Name: "name6"}}
	c7N = &CalculatedClient{Client: &Client{ID: "A7", Name: "name7"}}
)

func TestSortByNameAsc(t *testing.T) {
	// given
	a := []*CalculatedClient{c1N, c2N, c4N, c3N, c5N, c7N, c6N}

	// when
	SortByName(a, false)

	// then
	assert.ElementsMatch(t, a, []*CalculatedClient{c1N, c2N, c3N, c4N, c5N, c6N, c7N})
}

func TestSortByNameDesc(t *testing.T) {
	// given
	a := []*CalculatedClient{c1N, c2N, c4N, c3N, c5N, c7N, c6N}

	// when
	SortByName(a, false)

	// then
	assert.ElementsMatch(t, a, []*CalculatedClient{c7N, c6N, c5N, c4N, c3N, c2N, c1N})
}

var (
	c1OS = &CalculatedClient{Client: &Client{ID: "a1", OS: "OS12"}}
	c2OS = &CalculatedClient{Client: &Client{ID: "A2", OS: "os12"}}
	c3OS = &CalculatedClient{Client: &Client{ID: "a3", OS: "OS34"}}
	c4OS = &CalculatedClient{Client: &Client{ID: "A4", OS: "OS34"}}
	c5OS = &CalculatedClient{Client: &Client{ID: "a5", OS: "os5"}}
	c6OS = &CalculatedClient{Client: &Client{ID: "A6", OS: "OS6"}}
	c7OS = &CalculatedClient{Client: &Client{ID: "a7", OS: "os7"}}
)

func TestSortByOSAsc(t *testing.T) {
	// given
	a := []*CalculatedClient{c1OS, c2OS, c4OS, c3OS, c5OS, c7OS, c6OS}

	// when
	SortByOS(a, false)

	// then
	assert.ElementsMatch(t, a, []*CalculatedClient{c1OS, c2OS, c3OS, c4OS, c5OS, c6OS, c7OS})
}

func TestSortByOSDesc(t *testing.T) {
	// given
	a := []*CalculatedClient{c1OS, c2OS, c4OS, c3OS, c5OS, c7OS, c6OS}

	// when
	SortByOS(a, false)

	// then
	assert.ElementsMatch(t, a, []*CalculatedClient{c7OS, c6OS, c5OS, c4OS, c3OS, c2OS, c1OS})
}

var (
	c1H = &CalculatedClient{Client: &Client{ID: "A1", Hostname: "hostname12"}}
	c2H = &CalculatedClient{Client: &Client{ID: "a2", Hostname: "Hostname12"}}
	c3H = &CalculatedClient{Client: &Client{ID: "A3", Hostname: "hostname34"}}
	c4H = &CalculatedClient{Client: &Client{ID: "A4", Hostname: "HOSTNAME34"}}
	c5H = &CalculatedClient{Client: &Client{ID: "a5", Hostname: "hostname5"}}
	c6H = &CalculatedClient{Client: &Client{ID: "a6", Hostname: "hostname6"}}
	c7H = &CalculatedClient{Client: &Client{ID: "a7", Hostname: "Hostname7"}}
)

func TestSortByHostnameAsc(t *testing.T) {
	// given
	a := []*CalculatedClient{c1H, c2H, c4H, c3H, c5H, c7H, c6H}

	// when
	SortByHostname(a, false)

	// then
	assert.ElementsMatch(t, a, []*CalculatedClient{c1H, c2H, c3H, c4H, c5H, c6H, c7H})
}

func TestSortByHostnameDesc(t *testing.T) {
	// given
	a := []*CalculatedClient{c1H, c2H, c4H, c3H, c5H, c7H, c6H}

	// when
	SortByHostname(a, false)

	// then
	assert.ElementsMatch(t, a, []*CalculatedClient{c7H, c6H, c5H, c4H, c3H, c2H, c1H})
}
