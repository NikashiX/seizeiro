package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name           string
		senha          string
		fails          bool
		wantViolations []string
	}{
		{
			name:  "senha forte",
			senha: "Abc12345",
		},
		{
			name:  "senha forte (com caractere especial)",
			senha: "SenhaForte_2026",
		},
		{
			name:           "senha muito curta",
			senha:          "Ab1",
			fails:          true,
			wantViolations: []string{"entre 8 e 50 caracteres"},
		},
		{
			name:           "senha fraca (sem maiúscula e dígito)",
			senha:          "soletraminuscula",
			fails:          true,
			wantViolations: []string{"uma letra maiúscula", "um dígito"},
		},
		{
			name:           "senha fraca (sem minúscula)",
			senha:          "SOMAISCULA123",
			fails:          true,
			wantViolations: []string{"uma letra minúscula"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.senha)

			if !tt.fails && err != nil {
				t.Fatalf("expected not error for %q, got: %v", tt.senha, err)
			}
			if tt.fails && err == nil {
				t.Fatalf("expected error for %q", tt.senha)
			}

			if tt.fails {
				weakErr, ok := errors.AsType[*WeakPasswordError](err)
				if !ok {
					t.Fatalf("expected *WeakPasswordError, got %v", err)
				}

				if len(weakErr.Violations) != len(tt.wantViolations) {
					t.Errorf("invalid lenght. want %d, got %d", len(tt.wantViolations), len(weakErr.Violations))
					return
				}
			}
		})
	}
}
