// Package mailer é responsável pela infraestrutura de envio de e-mails da aplicação.
package mailer

import "context"

// Email representa uma mensagem de e-mail a ser enviada.
type Email struct {
	// To contém os endereços dos destinatários.
	To []string
	// Subject é o assunto da mensagem.
	Subject string
	// TextBody é o corpo da mensagem em texto puro.
	TextBody string
	// HTMLBody é o corpo alternativo em HTML. Opcional; quando vazio,
	// a mensagem é enviada apenas com o corpo em texto puro.
	HTMLBody string
}

// Mailer envia e-mails.
type Mailer interface {
	// Send envia o e-mail informado, retornando um erro caso o envio falhe.
	Send(ctx context.Context, email Email) error
}
