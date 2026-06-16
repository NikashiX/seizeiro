package wssei

import (
	"context"
	"net/http"
	"sync"
)

// Header que o WSSEI espera o token de autenticação.
const tokenHeader = "token"

// Implementação de [http.RoundTripper] que autentica no
// WSSEI usando um [Auth], injeta o token resultante no header de toda
// requisição e o reaproveita até que expire.
//
// O token é cacheado de forma thread-safe. Caso o servidor responda com
// 401/403, o cache é invalidado e uma nova autenticação é realizada
// automaticamente, repetindo a requisição original uma única vez.
type tokenTransport struct {
	http.RoundTripper
	auth    *Auth
	usuario string
	senha   string
	orgao   int

	mu          sync.Mutex
	cachedToken string
}

func (t *tokenTransport) next() http.RoundTripper {
	if t.RoundTripper != nil {
		return t.RoundTripper
	}
	return http.DefaultTransport
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	tok, err := t.token(ctx)
	if err != nil {
		return nil, err
	}

	res, err := t.do(req, tok)
	if err != nil {
		return nil, err
	}

	// Token expirado/inválido: invalida o cache, reautentica e tenta de novo.
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		res.Body.Close()

		tok, err = t.refreshToken(ctx, tok)
		if err != nil {
			return nil, err
		}

		return t.do(req, tok)
	}

	return res, nil
}

// do clona a requisição, injeta o header de token e a encaminha, sem mutar o
// request original.
func (t *tokenTransport) do(req *http.Request, tok string) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set(tokenHeader, tok)
	return t.next().RoundTrip(clone)
}

// Retorna o token cacheado ou autentica caso ainda não exista.
func (t *tokenTransport) token(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cachedToken != "" {
		return t.cachedToken, nil
	}

	auth, err := t.auth.Autenticar(ctx, t.usuario, t.senha, t.orgao)
	if err != nil {
		return "", err
	}

	t.cachedToken = auth.Token
	return auth.Token, nil
}

// Força uma nova autenticação, a menos que outra goroutine já
// tenha renovado o token enquanto aguardávamos o lock.
func (t *tokenTransport) refreshToken(ctx context.Context, stale string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cachedToken != "" && t.cachedToken != stale {
		return t.cachedToken, nil
	}

	auth, err := t.auth.Autenticar(ctx, t.usuario, t.senha, t.orgao)
	if err != nil {
		return "", err
	}

	t.cachedToken = auth.Token
	return auth.Token, nil
}
