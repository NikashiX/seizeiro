package wssei

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// newTestServer cria um servidor que responde /autenticar emitindo tokens
// sequenciais e registra, para o endpoint protegido, o último token recebido.
func newTestServer(t *testing.T) (srv *httptest.Server, authCount *int64, lastToken *atomic.Value) {
	t.Helper()

	var count int64
	var token atomic.Value
	token.Store("")

	mux := http.NewServeMux()
	mux.HandleFunc(apiBasePath+"/autenticar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("método inesperado em /autenticar: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type inesperado: %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if got := r.Form.Get("usuario"); got != "fulano" {
			t.Errorf("usuario = %q, esperado %q", got, "fulano")
		}
		if got := r.Form.Get("senha"); got != "segredo" {
			t.Errorf("senha = %q, esperado %q", got, "segredo")
		}
		if got := r.Form.Get("orgao"); got != "7" {
			t.Errorf("orgao = %q, esperado %q", got, "7")
		}

		n := atomic.AddInt64(&count, 1)
		w.Header().Set("Content-Type", "application/json")
		switch n {
		case 1:
			w.Write([]byte(`{"sucesso":true,"data":{"token":"token-1"}}`))
		default:
			w.Write([]byte(`{"sucesso":true,"data":{"token":"token-2"}}`))
		}
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &count, &token
}

func newTransport(endpoint string) *tokenTransport {
	return &tokenTransport{
		RoundTripper: http.DefaultTransport,
		auth:         NewAuth(endpoint),
		usuario:      "fulano",
		senha:        "segredo",
		orgao:        7,
	}
}

func TestTransport_InjectsToken(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)

	var got string
	mux := srv.Config.Handler.(*http.ServeMux)
	mux.HandleFunc("/protegido", func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(tokenHeader)
		w.WriteHeader(http.StatusOK)
	})

	client := &http.Client{Transport: newTransport(srv.URL)}

	res, err := client.Get(srv.URL + "/protegido")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	defer res.Body.Close()

	if got != "token-1" {
		t.Fatalf("header %q = %q, esperado %q", tokenHeader, got, "token-1")
	}
}

func TestTransport_DoesNotMutate(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)
	mux := srv.Config.Handler.(*http.ServeMux)
	mux.HandleFunc("/protegido", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/protegido", nil)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	tr := newTransport(srv.URL)
	res, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	defer res.Body.Close()

	if got := req.Header.Get(tokenHeader); got != "" {
		t.Fatalf("request original foi modificado, header = %q", got)
	}
}

func TestTransport_CachesToken(t *testing.T) {
	t.Parallel()

	srv, authCount, _ := newTestServer(t)
	mux := srv.Config.Handler.(*http.ServeMux)
	mux.HandleFunc("/protegido", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	client := &http.Client{Transport: newTransport(srv.URL)}

	for i := 0; i < 3; i++ {
		res, err := client.Get(srv.URL + "/protegido")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		res.Body.Close()
	}

	if n := atomic.LoadInt64(authCount); n != 1 {
		t.Fatalf("autenticações = %d, esperado 1 (token deveria ser cacheado)", n)
	}
}

func TestTransport_ReauthenticatesOnUnauthorized(t *testing.T) {
	t.Parallel()

	srv, authCount, _ := newTestServer(t)
	mux := srv.Config.Handler.(*http.ServeMux)

	var tokens []string
	mux.HandleFunc("/protegido", func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get(tokenHeader)
		tokens = append(tokens, tok)
		// O primeiro token é rejeitado; o segundo é aceito.
		if tok == "token-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	client := &http.Client{Transport: newTransport(srv.URL)}

	res, err := client.Get(srv.URL + "/protegido")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status final = %d, esperado 200", res.StatusCode)
	}
	if n := atomic.LoadInt64(authCount); n != 2 {
		t.Fatalf("autenticações = %d, esperado 2 (deveria reautenticar)", n)
	}
	if len(tokens) != 2 || tokens[0] != "token-1" || tokens[1] != "token-2" {
		t.Fatalf("tokens recebidos = %v, esperado [token-1 token-2]", tokens)
	}
}

func TestTransport_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"sucesso":false,"mensagem":"credenciais inválidas"}`))
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{Transport: newTransport(srv.URL)}

	_, err := client.Get(srv.URL + "/protegido")
	if err == nil {
		t.Fatal("esperado erro de autenticação, obteve nil")
	}
}
