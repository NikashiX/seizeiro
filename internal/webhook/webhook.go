// Package webhook é responsável por notificar receptores HTTP externos sobre
// eventos da aplicação (ex: cadastro de usuário do chatbot concluído,
// arquivo do SEI baixado).
//
// O envio é best-effort: falhas não interrompem o fluxo do chamador. Quando a
// URL configurada estiver vazia, todas as chamadas viram no-op.
//
// O [Notifier] usa uma URL base (sem o tipo de evento) e anexa o path
// específico de cada notificação (ex: "/cadastro", "/arquivo"). Assim, a
// configuração `CHATBOT_WEBHOOK_URL=http://localhost:3000/webhook` dispara
// para `/webhook/cadastro` ou `/webhook/arquivo` conforme o evento.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Header enviado pelo notifier quando há segredo configurado. O receptor deve
// comparar o valor com o segredo compartilhado antes de processar o payload.
//
// O nome `key` é o esperado pelo credential type "Header Auth" do n8n na
// configuração de produção e é replicado pelo whatsapp-sim em dev.
const secretHeader = "key"

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

// ArquivoPayload descreve um arquivo baixado do SEI, parte do [ArquivoEvent].
type ArquivoPayload struct {
	URL         string `json:"url"`
	Nome        string `json:"nome,omitempty"`
	ContentType string `json:"content_type"`
	Bytes       int64  `json:"bytes"`
	SHA256      string `json:"sha256"`
}

// ArquivoEvent é o payload enviado ao webhook quando o download de um anexo
// do SEI termina. Em caso de erro, [ArquivoEvent.Arquivo] vem nulo e
// [ArquivoEvent.Erro] contém a mensagem de falha.
type ArquivoEvent struct {
	Plataforma   string          `json:"plataforma"`
	PlataformaID string          `json:"plataforma_id"`
	Arquivo      *ArquivoPayload `json:"arquivo,omitempty"`
	Mensagem     string          `json:"mensagem,omitempty"`
	Erro         string          `json:"erro,omitempty"`
	OcorridoEm   string          `json:"ocorrido_em"`
}

// Notifier dispara notificações HTTP para receptores externos.
type Notifier struct {
	baseURL string
	secret  string
	http    *http.Client
}

// NewNotifier cria um [*Notifier] que envia POSTs para baseURL anexando o
// path do evento (ex: "/cadastro"). O secret, quando definido, é enviado no
// header `key`. Quando baseURL é vazia, o notifier vira no-op (útil em
// dev/local).
func NewNotifier(baseURL, secret string) *Notifier {
	return &Notifier{
		baseURL: strings.TrimRight(baseURL, "/"),
		secret:  secret,
		http:    &http.Client{Timeout: requestTimeout},
	}
}

func (n *Notifier) enabled() bool {
	return n != nil && n.baseURL != ""
}

// NotifyCadastro envia um POST com o evento de cadastro concluído. Falhas
// são apenas logadas.
func (n *Notifier) NotifyCadastro(ctx context.Context, event CadastroEvent) {
	if !n.enabled() {
		return
	}
	if event.OcorridoEm == "" {
		event.OcorridoEm = time.Now().UTC().Format(time.RFC3339)
	}
	n.notify(ctx, "cadastro", event)
}

// NotifyArquivo envia um POST com o evento de download de anexo
// (sucesso ou falha). Falhas no envio do webhook são apenas logadas.
func (n *Notifier) NotifyArquivo(ctx context.Context, event ArquivoEvent) {
	if !n.enabled() {
		return
	}
	if event.OcorridoEm == "" {
		event.OcorridoEm = time.Now().UTC().Format(time.RFC3339)
	}
	n.notify(ctx, "arquivo", event)
}

// notify executa o POST para `baseURL/path`. Internamente comum a todos os
// eventos.
func (n *Notifier) notify(ctx context.Context, path string, payload any) {
	url := n.baseURL + "/" + path

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook %s: marshal: %v", path, err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("webhook %s: new request: %v", path, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if n.secret != "" {
		req.Header.Set(secretHeader, n.secret)
	}

	res, err := n.http.Do(req)
	if err != nil {
		log.Printf("webhook %s: http do: %v", path, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		log.Printf("webhook %s: status %d: %s", path, res.StatusCode, bytes.TrimSpace(snippet))
		return
	}

	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		log.Printf("webhook %s: drain body: %v", path, err)
	}
}
