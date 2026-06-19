// Package webhook é responsável por notificar receptores HTTP externos sobre
// eventos da aplicação (ex: cadastro de usuário do chatbot concluído).
//
// O envio é best-effort: falhas não interrompem o fluxo do chamador. Quando a
// URL configurada estiver vazia, [Notifier.NotifyCadastro] vira no-op.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// Header enviado pelo notifier quando há segredo configurado. O receptor deve
// comparar o valor com o segredo compartilhado antes de processar o payload.
//
// O nome `key` é o esperado pelo credential type "Header Auth" do n8n na
// configuração de produção e é replicado pelo whatsapp-sim em dev.
const SecretHeader = "key"

// requestTimeout limita o tempo de cada chamada HTTP. Webhooks devem ser
// rápidos; quem precisa de processamento demorado deve responder logo e
// enfileirar o trabalho.
const requestTimeout = 5 * time.Second

// CadastroEvent é o payload enviado ao webhook quando um usuário do chatbot
// conclui o cadastro.
type CadastroEvent struct {
	Plataforma   string `json:"plataforma"`
	PlataformaID string `json:"plataforma_id"`
	SEIUsuario   string `json:"sei_usuario"`
	OcorridoEm   string `json:"ocorrido_em"`
}

// Notifier dispara notificações HTTP para receptores externos.
type Notifier struct {
	url    string
	secret string
	http   *http.Client
}

// NewNotifier cria um [*Notifier] que envia POSTs para url e injeta secret no
// header [SecretHeader] quando definido. Quando url é vazia, o notifier é
// criado mesmo assim e as chamadas viram no-op (útil em dev/local).
func NewNotifier(url, secret string) *Notifier {
	return &Notifier{
		url:    url,
		secret: secret,
		http:   &http.Client{Timeout: requestTimeout},
	}
}

// Enabled indica se o notifier vai efetivamente disparar requisições.
func (n *Notifier) Enabled() bool {
	return n != nil && n.url != ""
}

// NotifyCadastro envia um POST para a URL configurada com o evento de cadastro
// concluído. Falhas são apenas logadas: o cadastro do usuário não deve ser
// revertido por uma notificação que não chegou.
func (n *Notifier) NotifyCadastro(ctx context.Context, event CadastroEvent) {
	if !n.Enabled() {
		return
	}

	if event.OcorridoEm == "" {
		event.OcorridoEm = time.Now().UTC().Format(time.RFC3339)
	}

	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("webhook cadastro: marshal: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		log.Printf("webhook cadastro: new request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if n.secret != "" {
		req.Header.Set(SecretHeader, n.secret)
	}

	res, err := n.http.Do(req)
	if err != nil {
		log.Printf("webhook cadastro: http do: %v", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		// Lê um trecho do corpo para facilitar diagnóstico, mas limita o
		// tamanho para não logar páginas inteiras.
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		log.Printf("webhook cadastro: status %d: %s", res.StatusCode, bytes.TrimSpace(snippet))
		return
	}

	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		log.Printf("webhook cadastro: drain body: %v", err)
	}
}