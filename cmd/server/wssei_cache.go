package main

import (
	"sync"

	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
)

// wsseiClientCache mantém em memória clients [*wssei.Client] reaproveitáveis por
// usuário do chatbot. Como cada client carrega um [tokenTransport] que cacheia
// o token de autenticação do WSSEI, reaproveitar a instância evita um round-trip
// de login a cada chamada.
//
// A chave do cache é apenas (plataforma, plataforma_id); recadastros que mudam
// as credenciais SEI não são refletidos automaticamente — nesses casos a
// entrada precisa ser invalidada manualmente ou o processo reiniciado.
// O cache é seguro para uso concorrente.
type wsseiClientCache struct {
	baseURL string

	mu      sync.RWMutex
	entries map[wsseiClientCacheKey]*wssei.Client
}

// wsseiClientCacheKey identifica de forma única um usuário do chatbot no cache.
type wsseiClientCacheKey struct {
	Plataforma   string
	PlataformaID string
}

// newWSSEIClientCache cria um cache vazio para a [wssei.Config.BaseURL] informada.
func newWSSEIClientCache(baseURL string) *wsseiClientCache {
	return &wsseiClientCache{
		baseURL: baseURL,
		entries: make(map[wsseiClientCacheKey]*wssei.Client),
	}
}

// Get devolve o [*wssei.Client] correspondente ao usuário do chatbot,
// criando-o sob demanda.
func (c *wsseiClientCache) Get(usuario *chatbotauth.Usuario) *wssei.Client {
	key := wsseiClientCacheKey{
		Plataforma:   usuario.Plataforma,
		PlataformaID: usuario.PlataformaID,
	}

	c.mu.RLock()
	client, ok := c.entries[key]
	c.mu.RUnlock()
	if ok {
		return client
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Reverifica sob o lock para evitar criar duas instâncias em paralelo.
	if client, ok := c.entries[key]; ok {
		return client
	}

	client = wssei.NewClient(wssei.Config{
		BaseURL: c.baseURL,
		Usuario: usuario.SEIUsuario,
		Senha:   usuario.SEISenha,
		Orgao:   usuario.SEIOrgao,
	})
	c.entries[key] = client
	return client
}
