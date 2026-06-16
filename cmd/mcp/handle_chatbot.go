package main

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/automatiza-mg/seizeiro/internal/auth"
	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
	"github.com/danielgtaylor/huma/v2"
)

type CreateChatbotToken struct {
	Plataforma   string `json:"plataforma"`
	PlataformaID string `json:"plataforma_id"`
}

type CreateChatbotTokenRequest struct {
	Body CreateChatbotToken
}

type CreateChatbotTokenResponse struct {
	Body struct {
		CadastroURL string `json:"cadastro_url"`
	}
}

func registerCreateChatbotCadastro(api huma.API, pathPrefix string, baseURL string, service *chatbotauth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "chatbot-cadastrar",
		Method:      http.MethodPost,
		Path:        pathPrefix + "/chatbot/cadastros",
		Tags:        []string{"Chatbot"},
		Summary:     "Cria uma URL para cadastrar um novo usuário do chatbot",
	}, func(ctx context.Context, in *CreateChatbotTokenRequest) (*CreateChatbotTokenResponse, error) {
		token, err := service.CreateToken(ctx, in.Body.Plataforma, in.Body.PlataformaID)
		if err != nil {
			return nil, err
		}

		q := make(url.Values)
		q.Set("token", token.PlainText)
		cadastroURL := strings.TrimSuffix(baseURL, "/") + "/cadastro?" + q.Encode()

		var resp CreateChatbotTokenResponse
		resp.Body.CadastroURL = cadastroURL
		return &resp, nil
	})
}

func (app *application) handleCadastro(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	_, err := app.chatbotauth.GetTokenData(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidToken):
			http.Redirect(w, r, "/cadastro/invalido", http.StatusFound)
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	tmpl, err := template.New("cadastro.tmpl").ParseFS(app.views, "cadastro.tmpl")
	if err != nil {
		log.Printf("Server Error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	orgaos, err := app.scraper.ListOrgaos(r.Context())
	if err != nil {
		log.Printf("Server Error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, map[string]any{
		"Orgaos": orgaos,
	})
	if err != nil {
		log.Printf("Server Error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
}

func (app *application) handleCadastroPost(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	_, err := app.chatbotauth.GetTokenData(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidToken):
			http.Redirect(w, r, "/cadastro/invalido", http.StatusSeeOther)
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	usuario := r.FormValue("sei_usuario")
	senha := r.FormValue("sei_senha")
	orgaoStr := r.FormValue("sei_orgao")
	orgao, err := strconv.Atoi(orgaoStr)

	if usuario == "" || senha == "" || err != nil {
		http.Redirect(w, r, r.Referer(), http.StatusOK)
		return
	}

	_, err = wssei.NewAuth(app.cfg.SEI.BaseURL).Autenticar(r.Context(), usuario, senha, orgao)
	if err != nil {
		log.Printf("Failed to authenticate: %v", err)
		http.Redirect(w, r, r.Referer(), http.StatusOK)
		return
	}

	err = app.chatbotauth.CreateUsuario(r.Context(), chatbotauth.CreateUsuarioParams{
		Token:      token,
		SEIUsuario: usuario,
		SEISenha:   senha,
		SEIOrgao:   orgao,
	})
	if err != nil {
		log.Printf("Server Error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/cadastro/sucesso", http.StatusSeeOther)
}

func (app *application) handleCadastroSucesso(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("sucesso.tmpl").ParseFS(app.views, "sucesso.tmpl")
	if err != nil {
		log.Printf("Server Error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	if err != nil {
		log.Printf("Server Error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
}

func (app *application) handleCadastroTokenInvalido(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("invalido.tmpl").ParseFS(app.views, "invalido.tmpl")
	if err != nil {
		log.Printf("Server Error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	if err != nil {
		log.Printf("Server Error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
}
