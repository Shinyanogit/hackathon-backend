package ai

import "testing"

func TestParseCO2(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"basic int", "$40$", 40, false},
		{"decimal", "value is $12.5$", 12.5, false},
		{"no match", "nothing here", 0, true},
		{"multiple", "$1$ and $2$", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCO2(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("got=%v want=%v", got, tt.want)
			}
		})
	}
}
