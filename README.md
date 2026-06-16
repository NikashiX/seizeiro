# Seizeiro

Sistema de análise e triagem automática de processos SEI usando IA.

## Requerimentos

1. [Go 1.26](https://go.dev)
2. [PostgreSQL](https://www.postgresql.com)
3. [Docker](https://www.docker.com)
4. [goose](https://pressly.github.io/goose/)
5. [sqlc](https://sqlc.dev/)
6. [air](https://github.com/air-verse/air) - live reload durante o
   desenvolvimento, configurado em [`.air.toml`](.air.toml)

## Configuração

As configurações disponíveis podem ser encontradas no arquivo
[`.env.example`](.env.example) e são carregadas a partir de um arquivo `.env` na
raiz do projeto:

```bash
cp .env.example .env
```

## Infraestrutura de desenvolvimento

A infraestrutura local de desenvolvimento é definida no arquivo
[`compose.yml`](compose.yml). Para iniciar os serviços:

```bash
docker compose up -d
```

Para encerrá-los:

```bash
docker compose down
```

## Contribuindo

As instruções de contribuição estão disponíveis em [CONTRIBUTING.md](CONTRIBUTING.md).
