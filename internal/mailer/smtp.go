package mailer

import (
	"context"
	"fmt"
	"time"

	"github.com/wneessen/go-mail"
)

const (
	devPort = 1025
	sslPort = 465
)

var _ Mailer = (*SMTPMailer)(nil)

// SMTPMailer é uma implementação de [Mailer] que envia e-mails por SMTP.
type SMTPMailer struct {
	fromAddress string
	client      *mail.Client
}

// SMTPConfig contém as configurações de conexão com o servidor SMTP.
type SMTPConfig struct {
	// User e Password autenticam no servidor SMTP. Quando ambos estão
	// vazios, a conexão é feita sem autenticação.
	User     string
	Password string
	// Host é o endereço do servidor SMTP.
	Host string
	// Port é a porta do servidor SMTP e determina a política de TLS:
	// 1025 conecta sem TLS (desenvolvimento), 465 usa SSL/TLS implícito
	// e as demais exigem STARTTLS.
	Port int
	// FromAddress é o endereço usado como remetente das mensagens.
	FromAddress string
}

// NewSMTPMailer cria um novo [SMTPMailer] a partir da configuração informada.
func NewSMTPMailer(cfg SMTPConfig) (*SMTPMailer, error) {
	opts := []mail.Option{mail.WithPort(cfg.Port)}
	switch cfg.Port {
	case devPort:
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	case sslPort:
		opts = append(opts, mail.WithSSL())
	default:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	}

	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, err
	}
	if cfg.User != "" && cfg.Password != "" {
		client.SetSMTPAuth(mail.SMTPAuthAutoDiscover)
		client.SetUsername(cfg.User)
		client.SetPassword(cfg.Password)
	}

	return &SMTPMailer{
		fromAddress: cfg.FromAddress,
		client:      client,
	}, nil
}

// Send implementa [Mailer], enviando o e-mail pelo servidor SMTP configurado.
// O envio é limitado a um timeout de 15 segundos.
func (s *SMTPMailer) Send(ctx context.Context, email Email) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	msg := mail.NewMsg()
	msg.Subject(email.Subject)

	err := msg.From(s.fromAddress)
	if err != nil {
		return fmt.Errorf("msg from: %w", err)
	}

	err = msg.To(email.To...)
	if err != nil {
		return fmt.Errorf("msg to: %w", err)
	}

	msg.SetBodyString(mail.TypeTextPlain, email.TextBody)
	if email.HTMLBody != "" {
		msg.AddAlternativeString(mail.TypeTextHTML, email.HTMLBody)
	}

	if err := s.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("dial and send: %w", err)
	}
	return nil
}
