package service

import "testing"

// TestDefaultThresholdMatrix đối chiếu trực tiếp từng dòng của ma trận 18 tổ hợp
// với bảng trong BUSINESS-RULES.md mục 1.
func TestDefaultThresholdMatrix(t *testing.T) {
	cases := []struct {
		condition   string
		age         string
		sensitivity string
		want        int
	}{
		{"none", "child", "normal", 130},
		{"none", "child", "high", 110},
		{"none", "adult", "normal", 150},
		{"none", "adult", "high", 130},
		{"none", "elderly", "normal", 130},
		{"none", "elderly", "high", 110},

		{"asthma", "child", "normal", 80},
		{"asthma", "child", "high", 60},
		{"asthma", "adult", "normal", 100},
		{"asthma", "adult", "high", 80},
		{"asthma", "elderly", "normal", 80},
		{"asthma", "elderly", "high", 60},

		{"allergic_rhinitis", "child", "normal", 80},
		{"allergic_rhinitis", "child", "high", 60},
		{"allergic_rhinitis", "adult", "normal", 100},
		{"allergic_rhinitis", "adult", "high", 80},
		{"allergic_rhinitis", "elderly", "normal", 80},
		{"allergic_rhinitis", "elderly", "high", 60},
	}

	if len(defaultThresholdMatrix) != 18 {
		t.Fatalf("matrix must have exactly 18 combinations, got %d", len(defaultThresholdMatrix))
	}

	for _, c := range cases {
		got, err := defaultThresholdAQI(c.condition, c.age, c.sensitivity)
		if err != nil {
			t.Errorf("defaultThresholdAQI(%s,%s,%s) unexpectedly error: %v", c.condition, c.age, c.sensitivity, err)
			continue
		}
		if got != c.want {
			t.Errorf("defaultThresholdAQI(%s,%s,%s) = %d, want %d", c.condition, c.age, c.sensitivity, got, c.want)
		}
	}

	if _, err := defaultThresholdAQI("unknown", "adult", "normal"); err == nil {
		t.Error("expected error for unknown combination, got nil")
	}
}
