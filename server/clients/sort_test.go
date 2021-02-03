package clients

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortByIDAsc(t *testing.T) {
	// given
	a := []*Client{c1, c2, c3, c4}

	// when
	SortByID(a, false)

	// then
	assert.ElementsMatch(t, a, []*Client{c2, c4, c1, c3})
}

func TestSortByIDDesc(t *testing.T) {
	// given
	a := []*Client{c1, c2, c3, c4}

	// when
	SortByID(a, true)

	// then
	assert.ElementsMatch(t, a, []*Client{c3, c1, c4, c2})
}

var (
	c1N = &Client{ID: "1", Name: "name12"}
	c2N = &Client{ID: "2", Name: "name12"}
	c3N = &Client{ID: "3", Name: "name34"}
	c4N = &Client{ID: "4", Name: "name34"}
	c5N = &Client{ID: "5", Name: "name5"}
	c6N = &Client{ID: "6", Name: "name6"}
	c7N = &Client{ID: "7", Name: "name7"}
)

func TestSortByNameAsc(t *testing.T) {
	// given
	a := []*Client{c1N, c2N, c4N, c3N, c5N, c7N, c6N}

	// when
	SortByName(a, false)

	// then
	assert.ElementsMatch(t, a, []*Client{c1N, c2N, c3N, c4N, c5N, c6N, c7N})
}

func TestSortByNameDesc(t *testing.T) {
	// given
	a := []*Client{c1N, c2N, c4N, c3N, c5N, c7N, c6N}

	// when
	SortByName(a, false)

	// then
	assert.ElementsMatch(t, a, []*Client{c7N, c6N, c5N, c4N, c3N, c2N, c1N})
}

var (
	c1OS = &Client{ID: "1", OS: "OS12"}
	c2OS = &Client{ID: "2", OS: "OS12"}
	c3OS = &Client{ID: "3", OS: "OS34"}
	c4OS = &Client{ID: "4", OS: "OS34"}
	c5OS = &Client{ID: "5", OS: "OS5"}
	c6OS = &Client{ID: "6", OS: "OS6"}
	c7OS = &Client{ID: "7", OS: "OS7"}
)

func TestSortByOSAsc(t *testing.T) {
	// given
	a := []*Client{c1OS, c2OS, c4OS, c3OS, c5OS, c7OS, c6OS}

	// when
	SortByOS(a, false)

	// then
	assert.ElementsMatch(t, a, []*Client{c1OS, c2OS, c3OS, c4OS, c5OS, c6OS, c7OS})
}

func TestSortByOSDesc(t *testing.T) {
	// given
	a := []*Client{c1OS, c2OS, c4OS, c3OS, c5OS, c7OS, c6OS}

	// when
	SortByOS(a, false)

	// then
	assert.ElementsMatch(t, a, []*Client{c7OS, c6OS, c5OS, c4OS, c3OS, c2OS, c1OS})
}

var (
	c1H = &Client{ID: "1", Hostname: "hostname12"}
	c2H = &Client{ID: "2", Hostname: "hostname12"}
	c3H = &Client{ID: "3", Hostname: "hostname34"}
	c4H = &Client{ID: "4", Hostname: "hostname34"}
	c5H = &Client{ID: "5", Hostname: "hostname5"}
	c6H = &Client{ID: "6", Hostname: "hostname6"}
	c7H = &Client{ID: "7", Hostname: "hostname7"}
)

func TestSortByHostnameAsc(t *testing.T) {
	// given
	a := []*Client{c1H, c2H, c4H, c3H, c5H, c7H, c6H}

	// when
	SortByHostname(a, false)

	// then
	assert.ElementsMatch(t, a, []*Client{c1H, c2H, c3H, c4H, c5H, c6H, c7H})
}

func TestSortByHostnameDesc(t *testing.T) {
	// given
	a := []*Client{c1H, c2H, c4H, c3H, c5H, c7H, c6H}

	// when
	SortByHostname(a, false)

	// then
	assert.ElementsMatch(t, a, []*Client{c7H, c6H, c5H, c4H, c3H, c2H, c1H})
}
