// Package tasks contém os workers e args do River para jobs assíncronos.
//
// O padrão segue o projeto `automatiza`:
//
//   - Cada job define seu próprio `Args` + `InsertOpts` (queue, retries).
//   - O `Worker` embute `river.WorkerDefaults[Args]` e sobrescreve apenas
//     `Timeout` e `Work`.
//   - Erros determinísticos são envoltos em `river.JobCancel(err)` para
//     evitar retries inúteis; erros transitórios são retornados como erro
//     comum para o River reagendar.
//   - O padrão `lastAttempt := job.Attempt >= job.MaxAttempts` permite ao
//     worker tomar ação terminal antes do River desistir.
package tasks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/arquivo"
	"github.com/automatiza-mg/seizeiro/internal/webhook"
	"github.com/riverqueue/river"
)

// Constantes do job de download de anexo.
const (
	// BaixarAnexoMaxAttempts é o número máximo de tentativas no River.
	BaixarAnexoMaxAttempts = 3
	// BaixarAnexoTimeout é o limite de tempo para uma única execução.
	BaixarAnexoTimeout = 5 * time.Minute
)

// BaixarAnexoArgs é o payload do job de download.
//
// `Plataforma` + `PlataformaID` identificam o usuário do chatbot dono da
// requisição: usados para resolver as credenciais SEI e para rotear o
// webhook de notificação.
type BaixarAnexoArgs struct {
	Plataforma   string `json:"plataforma"`
	PlataformaID string `json:"plataforma_id"`
	IDProtocolo  int    `json:"id_protocolo"`
}

// Kind identifica o tipo de job no River.
func (BaixarAnexoArgs) Kind() string { return "arquivo:baixar-anexo" }

// InsertOpts concentra a política de enfileiramento. Quem chama
// `riverClient.Insert(...)` passa `opts = nil` e este método é consultado.
func (BaixarAnexoArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: BaixarAnexoMaxAttempts,
	}
}

// BaixarAnexoWorker processa [BaixarAnexoArgs]: resolve o cliente WSSEI do
// usuário, baixa o anexo via [arquivo.Service.Baixar] e dispara um webhook
// com o resultado (sucesso ou erro).
type BaixarAnexoWorker struct {
	river.WorkerDefaults[BaixarAnexoArgs]

	svc      *arquivo.Service
	notifier *webhook.Notifier
}

// NewBaixarAnexoWorker cria o worker com as dependências necessárias.
func NewBaixarAnexoWorker(svc *arquivo.Service, notifier *webhook.Notifier) *BaixarAnexoWorker {
	return &BaixarAnexoWorker{svc: svc, notifier: notifier}
}

// Timeout limita o tempo de uma única execução do job.
func (w *BaixarAnexoWorker) Timeout(*river.Job[BaixarAnexoArgs]) time.Duration {
	return BaixarAnexoTimeout
}

// Work executa o job. A política de retorno segue:
//
//   - usuário não cadastrado: [river.JobCancel] (não retentar, sem
//     destinatário válido para notificar).
//   - download bem-sucedido: dispara webhook de sucesso e retorna nil.
//   - download falhou e ainda há tentativas: retorna erro para o River
//     reagendar; nenhum webhook é enviado nas tentativas intermediárias.
//   - download falhou e foi a última tentativa: dispara webhook de erro e
//     retorna nil (evita retry adicional e dupla notificação).
func (w *BaixarAnexoWorker) Work(ctx context.Context, job *river.Job[BaixarAnexoArgs]) error {
	args := job.Args
	lastAttempt := job.Attempt >= job.MaxAttempts

	client, err := w.svc.WSSEIResolver().ResolveByPlataforma(ctx, args.Plataforma, args.PlataformaID)
	if err != nil {
		if errors.Is(err, arquivo.ErrUsuarioNotFound) {
			log.Printf("worker baixar_anexo: usuario nao cadastrado (%s/%s); cancelando job",
				args.Plataforma, args.PlataformaID)
			return river.JobCancel(err)
		}
		// Erro de infraestrutura ao resolver usuário: deixa o River reagendar.
		return fmt.Errorf("worker baixar_anexo: resolve usuario: %w", err)
	}

	res, execErr := w.svc.Baixar(ctx, client, args.IDProtocolo)
	if execErr != nil {
		if !lastAttempt {
			// Tenta de novo na próxima rodada sem notificar o usuário.
			return execErr
		}
		// Esgotou tentativas: notifica o usuário do erro definitivo.
		w.notifyErro(ctx, args, execErr)
		return nil
	}

	w.notifySucesso(ctx, args, res)
	return nil
}

// notifySucesso dispara o webhook de sucesso usando um contexto separado
// (best-effort). Se o ctx do job estiver cancelado/encerrado, ainda
// queremos tentar avisar o usuário dentro de um novo deadline.
func (w *BaixarAnexoWorker) notifySucesso(ctx context.Context, args BaixarAnexoArgs, res *arquivo.Resultado) {
	notifyCtx, cancel := context.WithTimeout(detach(ctx), 15*time.Second)
	defer cancel()

	w.notifier.NotifyArquivo(notifyCtx, webhook.ArquivoEvent{
		Plataforma:   args.Plataforma,
		PlataformaID: args.PlataformaID,
		Arquivo: &webhook.ArquivoPayload{
			URL:         res.URL,
			ContentType: res.ContentType,
			Bytes:       res.Bytes,
			SHA256:      res.Hash,
		},
	})
}

// notifyErro dispara o webhook de erro com a mensagem para o usuário.
func (w *BaixarAnexoWorker) notifyErro(ctx context.Context, args BaixarAnexoArgs, execErr error) {
	notifyCtx, cancel := context.WithTimeout(detach(ctx), 15*time.Second)
	defer cancel()

	log.Printf("worker baixar_anexo: falha definitiva (%s/%s id=%d): %v",
		args.Plataforma, args.PlataformaID, args.IDProtocolo, execErr)

	w.notifier.NotifyArquivo(notifyCtx, webhook.ArquivoEvent{
		Plataforma:   args.Plataforma,
		PlataformaID: args.PlataformaID,
		Erro:         execErr.Error(),
	})
}

// detach devolve um contexto novo que preserva valores mas não é
// cancelado quando o pai cancela. Usado para tarefas de "limpeza/aviso"
// (webhook) que devem rodar mesmo após o ctx do job terminar.
func detach(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}
