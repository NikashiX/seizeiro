package notify

import "testing"

func TestNewLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		clientURL string
		fails     bool
	}{
		{
			name:      "url válida",
			clientURL: "https://app.example.com",
		},
		{
			name:      "url válida (com barra no final)",
			clientURL: "https://app.example.com/",
		},
		{
			name:      "url válida (com prefixo de caminho)",
			clientURL: "https://example.com/app",
		},
		{
			name:      "url válida (http em desenvolvimento)",
			clientURL: "http://localhost:5173",
		},
		{
			name:      "url vazia",
			clientURL: "",
			fails:     true,
		},
		{
			name:      "url relativa",
			clientURL: "/app",
			fails:     true,
		},
		{
			name:      "url sem esquema",
			clientURL: "app.example.com",
			fails:     true,
		},
		{
			name:      "url inválida",
			clientURL: "https://app.exa mple.com",
			fails:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links, err := NewLinks(tt.clientURL)

			if !tt.fails && err != nil {
				t.Fatalf("expected no error for %q, got: %v", tt.clientURL, err)
			}
			if tt.fails && err == nil {
				t.Fatalf("expected error for %q", tt.clientURL)
			}
			if !tt.fails && links == nil {
				t.Fatalf("expected links for %q, got nil", tt.clientURL)
			}
		})
	}
}

func TestLinks_AtivarConta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		clientURL string
		token     string
		want      string
	}{
		{
			name:      "url base sem barra no final",
			clientURL: "https://app.example.com",
			token:     "abc123",
			want:      "https://app.example.com/ativar-conta?token=abc123",
		},
		{
			name:      "url base com barra no final",
			clientURL: "https://app.example.com/",
			token:     "abc123",
			want:      "https://app.example.com/ativar-conta?token=abc123",
		},
		{
			name:      "url base com prefixo de caminho",
			clientURL: "https://example.com/app",
			token:     "abc123",
			want:      "https://example.com/app/ativar-conta?token=abc123",
		},
		{
			name:      "token com caracteres especiais é escapado",
			clientURL: "https://app.example.com",
			token:     "a b+c&d=e/f",
			want:      "https://app.example.com/ativar-conta?token=a+b%2Bc%26d%3De%2Ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links, err := NewLinks(tt.clientURL)
			if err != nil {
				t.Fatalf("NewLinks(%q): %v", tt.clientURL, err)
			}

			got := links.AtivarConta(tt.token)
			if got != tt.want {
				t.Errorf("AtivarConta(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

func TestLinks_RedefinirSenha(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		clientURL string
		token     string
		want      string
	}{
		{
			name:      "url base sem barra no final",
			clientURL: "https://app.example.com",
			token:     "abc123",
			want:      "https://app.example.com/redefinir-senha?token=abc123",
		},
		{
			name:      "url base com prefixo de caminho",
			clientURL: "https://example.com/app/",
			token:     "abc123",
			want:      "https://example.com/app/redefinir-senha?token=abc123",
		},
		{
			name:      "token com caracteres especiais é escapado",
			clientURL: "https://app.example.com",
			token:     "tok en?&",
			want:      "https://app.example.com/redefinir-senha?token=tok+en%3F%26",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links, err := NewLinks(tt.clientURL)
			if err != nil {
				t.Fatalf("NewLinks(%q): %v", tt.clientURL, err)
			}

			got := links.RedefinirSenha(tt.token)
			if got != tt.want {
				t.Errorf("RedefinirSenha(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

// Garante que os métodos de link não modificam a URL base entre chamadas.
func TestLinks_BaseNotMutated(t *testing.T) {
	t.Parallel()

	links, err := NewLinks("https://app.example.com")
	if err != nil {
		t.Fatalf("NewLinks: %v", err)
	}

	first := links.AtivarConta("tok1")
	_ = links.RedefinirSenha("tok2")
	second := links.AtivarConta("tok1")

	if first != second {
		t.Errorf("link changed between calls: %q != %q", first, second)
	}
}
