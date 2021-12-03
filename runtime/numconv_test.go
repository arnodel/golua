package runtime

import (
	"reflect"
	"testing"
)

func TestToNumber(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		n    int64
		x    float64
		tp   NumberType
	}{
		{
			name: "int",
			v:    IntValue(23),
			n:    23,
			tp:   IsInt,
		},
		{
			name: "float",
			v:    FloatValue(1.1),
			x:    1.1,
			tp:   IsFloat,
		},
		{
			name: "int string",
			v:    StringValue("-12"),
			n:    -12,
			tp:   IsInt,
		},
		{
			name: "float string",
			v:    StringValue("1.45"),
			x:    1.45,
			tp:   IsFloat,
		},
		{
			name: "non numeric string",
			v:    StringValue("hello"),
			tp:   NaN,
		},
		{
			name: "string with numeric prefix",
			v:    StringValue("49ers"),
			tp:   NaN,
		},
		{
			name: "boolean",
			v:    BoolValue(false),
			tp:   NaN,
		},
		{
			name: "nil",
			v:    NilValue,
			tp:   NaN,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := ToNumber(tt.v)
			if got != tt.n {
				t.Errorf("ToNumber() got = %v, want %v", got, tt.n)
			}
			if got1 != tt.x {
				t.Errorf("ToNumber() got1 = %v, want %v", got1, tt.x)
			}
			if got2 != tt.tp {
				t.Errorf("ToNumber() got2 = %v, want %v", got2, tt.tp)
			}
		})
	}
}

func TestToNumberValue(t *testing.T) {

	tests := []struct {
		name string
		v    Value
		want Value
		tp   NumberType
	}{
		{
			name: "int",
			v:    IntValue(23),
			want: IntValue(23),
			tp:   IsInt,
		},
		{
			name: "float",
			v:    FloatValue(1.1),
			want: FloatValue(1.1),
			tp:   IsFloat,
		},
		{
			name: "int string",
			v:    StringValue("-12"),
			want: IntValue(-12),
			tp:   IsInt,
		},
		{
			name: "float string",
			v:    StringValue("1.45"),
			want: FloatValue(1.45),
			tp:   IsFloat,
		},
		{
			name: "non numeric string",
			v:    StringValue("hello"),
			tp:   NaN,
		},
		{
			name: "string with numeric prefix",
			v:    StringValue("49ers"),
			tp:   NaN,
		},
		{
			name: "boolean",
			v:    BoolValue(false),
			tp:   NaN,
		},
		{
			name: "nil",
			v:    NilValue,
			tp:   NaN,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ToNumberValue(tt.v)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToNumberValue() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.tp {
				t.Errorf("ToNumberValue() got1 = %v, want %v", got1, tt.tp)
			}
		})
	}
}

func TestToInt(t *testing.T) {

	tests := []struct {
		name string
		v    Value
		want int64
		ok   bool
	}{
		{
			name: "int",
			v:    IntValue(100),
			want: 100,
			ok:   true,
		},
		{
			name: "integral float",
			v:    FloatValue(53),
			want: 53,
			ok:   true,
		},
		{
			name: "int string",
			v:    StringValue("1e6"),
			want: 1e6,
			ok:   true,
		},
		{
			name: "decimal float",
			v:    FloatValue(2.3),
		},
		{
			name: "decimal string",
			v:    StringValue("55.5"),
		},
		{
			name: "nil",
		},
		{
			name: "boolean",
			v:    BoolValue(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ToInt(tt.v)
			if got != tt.want {
				t.Errorf("ToInt() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("ToInt() got1 = %v, want %v", got1, tt.ok)
			}
		})
	}
}

func TestToIntNoString(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		want int64
		ok   bool
	}{
		{
			name: "int",
			v:    IntValue(100),
			want: 100,
			ok:   true,
		},
		{
			name: "integral float",
			v:    FloatValue(53),
			want: 53,
			ok:   true,
		},
		{
			name: "int string",
			v:    StringValue("1e6"),
		},
		{
			name: "decimal float",
			v:    FloatValue(2.3),
		},
		{
			name: "decimal string",
			v:    StringValue("55.5"),
		},
		{
			name: "nil",
		},
		{
			name: "boolean",
			v:    BoolValue(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ToIntNoString(tt.v)
			if got != tt.want {
				t.Errorf("ToIntNoString() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("ToIntNoString() got1 = %v, want %v", got1, tt.ok)
			}
		})
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		want float64
		ok   bool
	}{
		{
			name: "int",
			v:    IntValue(100),
			want: 100,
			ok:   true,
		},
		{
			name: "integral float",
			v:    FloatValue(53),
			want: 53,
			ok:   true,
		},
		{
			name: "int string",
			v:    StringValue("1e6"),
			want: 1e6,
			ok:   true,
		},
		{
			name: "decimal float",
			v:    FloatValue(2.3),
			want: 2.3,
			ok:   true,
		},
		{
			name: "decimal string",
			v:    StringValue("55.5"),
			want: 55.5,
			ok:   true,
		},
		{
			name: "nil",
		},
		{
			name: "boolean",
			v:    BoolValue(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ToFloat(tt.v)
			if got != tt.want {
				t.Errorf("ToFloat() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.ok {
				t.Errorf("ToFloat() got1 = %v, want %v", got1, tt.ok)
			}
		})
	}
}
