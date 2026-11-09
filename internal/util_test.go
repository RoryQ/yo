package internal

import "testing"

func TestArrayElemLen(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		want     int
	}{
		{
			name:     "StringArray_ReturnsElementLength",
			dataType: "ARRAY<STRING(100)>",
			want:     100,
		},
		{
			name:     "BytesArray_ReturnsElementLength",
			dataType: "ARRAY<BYTES(65)>",
			want:     65,
		},
		{
			name:     "SingleDigit_ReturnsElementLength",
			dataType: "ARRAY<STRING(1)>",
			want:     1,
		},
		{
			name:     "StringArrayMax_ReturnsMinusOne",
			dataType: "ARRAY<STRING(MAX)>",
			want:     -1,
		},
		{
			name:     "BytesArrayMax_ReturnsMinusOne",
			dataType: "ARRAY<BYTES(MAX)>",
			want:     -1,
		},
		{
			name:     "ArrayWithoutLength_ReturnsMinusOne",
			dataType: "ARRAY<INT64>",
			want:     -1,
		},
		{
			name:     "ScalarString_ReturnsMinusOne",
			dataType: "STRING(32)",
			want:     -1,
		},
		{
			name:     "Empty_ReturnsMinusOne",
			dataType: "",
			want:     -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := arrayElemLen(tt.dataType); got != tt.want {
				t.Errorf("arrayElemLen(%q) = %d, want %d", tt.dataType, got, tt.want)
			}
		})
	}
}
