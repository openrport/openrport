package sessions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortByIDAsc(t *testing.T) {
	// given
	a := []*ClientSession{s1, s2, s3, s4}

	// when
	SortByID(a, false)

	// then
	assert.ElementsMatch(t, a, []*ClientSession{s2, s4, s1, s3})
}

func TestSortByIDDesc(t *testing.T) {
	// given
	a := []*ClientSession{s1, s2, s3, s4}

	// when
	SortByID(a, true)

	// then
	assert.ElementsMatch(t, a, []*ClientSession{s3, s1, s4, s2})
}

var (
	s1N = &ClientSession{ID: "1", Name: "name12"}
	s2N = &ClientSession{ID: "2", Name: "name12"}
	s3N = &ClientSession{ID: "3", Name: "name34"}
	s4N = &ClientSession{ID: "4", Name: "name34"}
	s5N = &ClientSession{ID: "5", Name: "name5"}
	s6N = &ClientSession{ID: "6", Name: "name6"}
	s7N = &ClientSession{ID: "7", Name: "name7"}
)

func TestSortByNameAsc(t *testing.T) {
	// given
	a := []*ClientSession{s1N, s2N, s4N, s3N, s5N, s7N, s6N}

	// when
	SortByName(a, false)

	// then
	assert.ElementsMatch(t, a, []*ClientSession{s1N, s2N, s3N, s4N, s5N, s6N, s7N})
}

func TestSortByNameDesc(t *testing.T) {
	// given
	a := []*ClientSession{s1N, s2N, s4N, s3N, s5N, s7N, s6N}

	// when
	SortByName(a, false)

	// then
	assert.ElementsMatch(t, a, []*ClientSession{s7N, s6N, s5N, s4N, s3N, s2N, s1N})
}

var (
	s1OS = &ClientSession{ID: "1", OS: "OS12"}
	s2OS = &ClientSession{ID: "2", OS: "OS12"}
	s3OS = &ClientSession{ID: "3", OS: "OS34"}
	s4OS = &ClientSession{ID: "4", OS: "OS34"}
	s5OS = &ClientSession{ID: "5", OS: "OS5"}
	s6OS = &ClientSession{ID: "6", OS: "OS6"}
	s7OS = &ClientSession{ID: "7", OS: "OS7"}
)

func TestSortByOSAsc(t *testing.T) {
	// given
	a := []*ClientSession{s1OS, s2OS, s4OS, s3OS, s5OS, s7OS, s6OS}

	// when
	SortByOS(a, false)

	// then
	assert.ElementsMatch(t, a, []*ClientSession{s1OS, s2OS, s3OS, s4OS, s5OS, s6OS, s7OS})
}

func TestSortByOSDesc(t *testing.T) {
	// given
	a := []*ClientSession{s1OS, s2OS, s4OS, s3OS, s5OS, s7OS, s6OS}

	// when
	SortByOS(a, false)

	// then
	assert.ElementsMatch(t, a, []*ClientSession{s7OS, s6OS, s5OS, s4OS, s3OS, s2OS, s1OS})
}

var (
	s1H = &ClientSession{ID: "1", Hostname: "hostname12"}
	s2H = &ClientSession{ID: "2", Hostname: "hostname12"}
	s3H = &ClientSession{ID: "3", Hostname: "hostname34"}
	s4H = &ClientSession{ID: "4", Hostname: "hostname34"}
	s5H = &ClientSession{ID: "5", Hostname: "hostname5"}
	s6H = &ClientSession{ID: "6", Hostname: "hostname6"}
	s7H = &ClientSession{ID: "7", Hostname: "hostname7"}
)

func TestSortByHostnameAsc(t *testing.T) {
	// given
	a := []*ClientSession{s1H, s2H, s4H, s3H, s5H, s7H, s6H}

	// when
	SortByHostname(a, false)

	// then
	assert.ElementsMatch(t, a, []*ClientSession{s1H, s2H, s3H, s4H, s5H, s6H, s7H})
}

func TestSortByHostnameDesc(t *testing.T) {
	// given
	a := []*ClientSession{s1H, s2H, s4H, s3H, s5H, s7H, s6H}

	// when
	SortByHostname(a, false)

	// then
	assert.ElementsMatch(t, a, []*ClientSession{s7H, s6H, s5H, s4H, s3H, s2H, s1H})
}
