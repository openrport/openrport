package dynops_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/realvnc-labs/rport/share/dynops"
	"github.com/realvnc-labs/rport/share/dynops/dyncopy"
	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/random"
)

type TestStruct struct {
	FieldA    string
	FieldB    int
	FieldC    []string
	FieldTime time.Time
}

func TestPaginator(t *testing.T) {
	type args[T any] struct {
		list       []T
		pagination *query.Pagination
	}
	type testCase[T any] struct {
		name string
		args args[T]
		want []T
	}
	tests := []testCase[TestStruct]{
		{
			name: "test empty",
			args: args[TestStruct]{
				list:       nil,
				pagination: nil,
			},
			want: nil,
		},
		{
			name: "test empty but non nil",
			args: args[TestStruct]{
				list: []TestStruct{},
				pagination: &query.Pagination{
					Limit:           "",
					Offset:          "",
					ValidatedLimit:  0,
					ValidatedOffset: 0,
				},
			},
			want: []TestStruct{},
		},
		{
			name: "test simple limit",
			args: args[TestStruct]{
				list: []TestStruct{{}, {
					FieldA: "A",
				}, {}},
				pagination: &query.Pagination{
					Limit:           "",
					Offset:          "",
					ValidatedLimit:  1,
					ValidatedOffset: 1,
				},
			},
			want: []TestStruct{{
				FieldA: "A",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dynops.Paginator(tt.args.list, tt.args.pagination); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Paginator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSorter(t *testing.T) {
	type args[T any] struct {
		list  []T
		sorts []query.SortOption
	}
	type testCase[T any] struct {
		name string
		args args[T]
		want []T
	}
	tests := []testCase[TestStruct]{
		{
			name: "test empty",
			args: args[TestStruct]{
				list:  nil,
				sorts: nil,
			},
			want: nil,
		},
		{
			name: "test empty but non nil",
			args: args[TestStruct]{
				list:  []TestStruct{},
				sorts: []query.SortOption{},
			},
			want: []TestStruct{},
		},
		{
			name: "test simple sort",
			args: args[TestStruct]{
				list: []TestStruct{
					{FieldA: "B"}, {FieldA: "A"}, {FieldA: "C"},
				},
				sorts: []query.SortOption{
					{
						Column: "FieldA",
						IsASC:  false,
					},
				},
			},
			want: []TestStruct{{FieldA: "A"}, {FieldA: "B"}, {FieldA: "C"}},
		},
		{
			name: "test nested reverse sort",
			args: args[TestStruct]{
				list: []TestStruct{
					{FieldA: "B", FieldB: 1}, {FieldA: "A"}, {FieldA: "B", FieldB: 2},
				},
				sorts: []query.SortOption{
					{
						Column: "FieldA",
						IsASC:  false,
					},
					{
						Column: "FieldB",
						IsASC:  true,
					},
				},
			},
			want: []TestStruct{{FieldA: "A"}, {FieldA: "B", FieldB: 2}, {FieldA: "B", FieldB: 1}},
		},
		{
			name: "test int sort",
			args: args[TestStruct]{
				list: []TestStruct{
					{FieldA: "B", FieldB: 11}, {FieldA: "A"}, {FieldA: "B", FieldB: 2},
				},
				sorts: []query.SortOption{
					{
						Column: "FieldB",
						IsASC:  false,
					},
				},
			},
			want: []TestStruct{{FieldA: "A"}, {FieldA: "B", FieldB: 2}, {FieldA: "B", FieldB: 11}},
		},
		{
			name: "test time sort",
			args: args[TestStruct]{
				list: []TestStruct{
					{FieldTime: time.Time{}, FieldA: "A"}, {FieldTime: time.Time{}.Add(time.Hour), FieldA: "B"},
				},
				sorts: []query.SortOption{
					{
						Column: "FieldTime",
						IsASC:  true,
					},
				},
			},
			want: []TestStruct{{FieldTime: time.Time{}.Add(time.Hour), FieldA: "B"}, {FieldTime: time.Time{}, FieldA: "A"}},
		},
		{
			name: "test nested reverse sort",
			args: args[TestStruct]{
				list: []TestStruct{
					{FieldA: "B", FieldC: []string{"b", "a"}}, {FieldA: "A"}, {FieldA: "B", FieldC: []string{"a", "b"}},
				},
				sorts: []query.SortOption{
					{
						Column: "FieldA",
						IsASC:  false,
					},
					{
						Column: "FieldC",
						IsASC:  true,
					},
				},
			},
			want: []TestStruct{{FieldA: "A"}, {FieldA: "B", FieldC: []string{"b", "a"}}, {FieldA: "B", FieldC: []string{"a", "b"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttt := dyncopy.BuildTranslationTable(TestStruct{})
			if got, _ := dynops.FastSorter1(ttt, tt.args.list, tt.args.sorts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlowBasicSorter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func MakeList(N int) []TestStruct {
	list := make([]TestStruct, N)
	for i := 0; i < N; i++ {
		list[i].FieldB = rand.Int()
		list[i].FieldA = random.String(3, "asdfghjkl;'qwertyuiozxcvn,m123457809")
	}
	return list
}

var table = []struct {
	input int
}{
	{input: 100},
	{input: 1000},
	{input: 10000},
	{input: 100000},
}

var GenList = MakeList(table[len(table)-1].input)

func BenchmarkSortSpeed(b *testing.B) {
	for _, v := range table {
		b.Run(fmt.Sprintf("input_size_%d", v.input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				dynops.SlowBasicSorter(GenList[:v.input], []query.SortOption{
					{
						Column: "FieldA",
						IsASC:  false,
					},
					{
						Column: "FieldB",
						IsASC:  true,
					},
				})
			}
		})
	}
}

var tt = dyncopy.BuildTranslationTable(GenList[0])

func BenchmarkFastSort1Speed(b *testing.B) {
	for _, v := range table {
		b.Run(fmt.Sprintf("input_size_%d", v.input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = dynops.FastSorter1(tt, GenList[:v.input], []query.SortOption{
					{
						Column: "FieldA",
						IsASC:  false,
					},
					{
						Column: "FieldB",
						IsASC:  true,
					},
				})
			}
		})
	}
}
