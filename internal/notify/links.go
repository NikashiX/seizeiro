package notify

import (
	"fmt"
	"net/url"
)

// Rotas do frontend usadas na montagem dos links enviados por e-mail.
const (
	// AtivarContaPath é a rota do frontend de ativação de conta.
	AtivarContaPath = "/ativar-conta"
	// RedefinirSenhaPath é a rota do frontend de redefinição de senha.
	RedefinirSenhaPath = "/redefinir-senha"
)

// Links monta as URLs do frontend enviadas por e-mail (ativação de conta,
// redefinição de senha, etc.) a partir de uma URL base.
type Links struct {
	base *url.URL
}

// NewLinks cria um novo [Links] a partir da URL pública.
func NewLinks(clientURL string) (*Links, error) {
	base, err := url.Parse(clientURL)
	if err != nil {
		return nil, fmt.Errorf("url parse: %w", err)
	}
	if !base.IsAbs() {
		return nil, fmt.Errorf("url is not absolute: %q", base)
	}

	return &Links{
		base: base,
	}, nil
}

// withToken junta path à URL base e adiciona o token como query param.
func (l *Links) withToken(path, token string) string {
	q := make(url.Values)
	q.Set("token", token)

	u := l.base.JoinPath(path)
	u.RawQuery = q.Encode()

	return u.String()
}

// AtivarConta retorna o link de ativação de conta com o token informado.
func (l *Links) AtivarConta(token string) string {
	return l.withToken(AtivarContaPath, token)
}

// RedefinirSenha retorna o link de redefinição de senha com o token informado.
func (l *Links) RedefinirSenha(token string) string {
	return l.withToken(RedefinirSenhaPath, token)
}
