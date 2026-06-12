// Package notify é responsável por gerenciar notificações de diversos canais.
package notify

import (
	"context"
	"fmt"

	"github.com/automatiza-mg/seizeiro/internal/mailer"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type Notifier struct {
	links *Links
	river *river.Client[pgx.Tx]
}

func NewNotifier(clientURL string, riverClient *river.Client[pgx.Tx]) (*Notifier, error) {
	links, err := NewLinks(clientURL)
	if err != nil {
		return nil, err
	}

	return &Notifier{
		links: links,
		river: riverClient,
	}, nil
}

func (n *Notifier) SendAtivarConta(ctx context.Context, tx pgx.Tx, emailAddress, token string) error {
	link := n.links.AtivarConta(token)

	args := mailer.SendEmailArgs{
		Email: mailer.Email{
			To:       []string{emailAddress},
			Subject:  "Ativar Conta",
			TextBody: fmt.Sprintf("Sua conta foi criado com sucesso!\nClique no link ao lado para ativá-la: %s", link),
		},
	}

	_, err := n.insertTask(ctx, tx, args)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

func (n *Notifier) insertTask(ctx context.Context, tx pgx.Tx, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	if tx == nil {
		return n.river.Insert(ctx, args, nil)
	}
	return n.river.InsertTx(ctx, tx, args, nil)
}
