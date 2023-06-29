package refs

import (
	"reflect"
	"testing"
)

const TestType IdentifiableType = "test-type"

func TestNewIdentifiable(t *testing.T) {
	type args struct {
		iType IdentifiableType
		id    string
	}
	tests := []struct {
		name string
		args args
		want identifiable
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewIdentifiable(tt.args.iType, tt.args.id); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewIdentifiable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseIdentifiable(t *testing.T) {
	type args struct {
		raw string
	}
	tests := []struct {
		name    string
		args    args
		want    Identifiable
		wantErr bool
	}{
		{
			name: "test bad parse",
			args: args{
				raw: "bad-text",
			},
			want:    identifiable{},
			wantErr: true,
		},
		{
			name: "test good parse",
			args: args{
				raw: "bad-text:::some-id",
			},
			want: identifiable{
				iType: "bad-text",
				id:    "some-id",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIdentifiable(tt.args.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIdentifiable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseIdentifiable() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_identifiable_ID(t *testing.T) {
	type fields struct {
		iType IdentifiableType
		id    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "check simple serialization",
			fields: fields{
				iType: TestType,
				id:    "some-weird::id",
			},
			want: "some-weird::id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := identifiable{
				iType: tt.fields.iType,
				id:    tt.fields.id,
			}
			if got := i.ID(); got != tt.want {
				t.Errorf("ID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_identifiable_String(t *testing.T) {
	type fields struct {
		iType IdentifiableType
		id    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "check simple serialization",
			fields: fields{
				iType: TestType,
				id:    "some-weird::id",
			},
			want: "test-type:::some-weird::id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := identifiable{
				iType: tt.fields.iType,
				id:    tt.fields.id,
			}
			if got := i.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_identifiable_Type(t *testing.T) {
	type fields struct {
		iType IdentifiableType
		id    string
	}
	tests := []struct {
		name   string
		fields fields
		want   IdentifiableType
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := identifiable{
				iType: tt.fields.iType,
				id:    tt.fields.id,
			}
			if got := i.Type(); got != tt.want {
				t.Errorf("Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMustIdentifiableFactory(t *testing.T) {
	type args struct {
		notificationType IdentifiableType
	}
	tests := []struct {
		name string
		args args
		want identifiable
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustIdentifiableFactory(tt.args.notificationType); !reflect.DeepEqual(got("some-id"), tt.want) {
				t.Errorf("MustIdentifiableFactory() = %v, want %v", got("some-id"), tt.want)
			}
		})
	}
}
