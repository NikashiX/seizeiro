package mailer

import (
	"context"

	"github.com/riverqueue/river"
)

// SendEmailArgs são os argumentos do job de envio de e-mail processado
// por [Worker]. Insira um job com esses argumentos no River para enviar
// um e-mail de forma assíncrona.
type SendEmailArgs struct {
	Email Email `json:"email"`
}

// Kind implementa [river.JobArgs].
func (args SendEmailArgs) Kind() string {
	return "email:send"
}

// Worker é o worker do River que processa jobs de [SendEmailArgs],
// enviando os e-mails por meio de um [Mailer].
type Worker struct {
	mailer Mailer
	river.WorkerDefaults[SendEmailArgs]
}

// NewWorker cria um novo [Worker] que envia e-mails usando o [Mailer] informado.
func NewWorker(mailer Mailer) *Worker {
	return &Worker{
		mailer: mailer,
	}
}

// Work implementa [river.Worker], enviando o e-mail contido nos argumentos do job.
func (w *Worker) Work(ctx context.Context, job *river.Job[SendEmailArgs]) error {
	return w.mailer.Send(ctx, job.Args.Email)
}
