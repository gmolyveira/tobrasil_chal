# Desafio DevOps 2025 — T.O Brasil

Duas aplicações web em **linguagens diferentes** (Python/FastAPI e Go/net/http), atrás de **Nginx** como reverse proxy com **cache com TTLs distintos**, orquestradas com **Docker Compose**, com **observabilidade** (Prometheus, Grafana, nginx-prometheus-exporter) e **dashboards** provisionados.

## Diagrama da arquitetura

![Arquitetura da stack: cliente, Nginx, apps, exporter, Prometheus e Grafana](to_brasil_arquitetura.png)


**Fluxo de métricas do Nginx:** o **nginx-prometheus-exporter** faz *pull* HTTP ao endpoint **`stub_status`** do Nginx; o **Prometheus** faz scrape do exporter (e das apps em `/metrics`).

## Pré-requisitos

- Docker e Docker Compose (plugin `compose` v2)

## Como executar

Um comando sobe toda a infra:

```bash
docker compose up --build -d
```

## Endpoints HTTP

| Rota | Descrição | Cache (Nginx) |
|---|---|---|
| `http://localhost/python/` | Texto fixo (Python) | 10s |
| `http://localhost/python/time` | Horário do servidor (Python) | 10s |
| `http://localhost/go/` | Texto fixo (Go) | 60s |
| `http://localhost/go/time` | Horário do servidor (Go) | 60s |

O header **`X-Cache-Status`** indica `HIT`, `MISS`, `STALE`, etc.

## Observabilidade

| Serviço | URL | Papel |
|---|---|---|
| Prometheus | http://localhost:9090 | Coleta de métricas (scrape) |
| Grafana | http://localhost:3000 | Dashboards (usuário/senha padrão: **admin/admin** — trocar em ambiente real) |

O Prometheus coleta:

- **app-python** — `/metrics` via `prometheus-fastapi-instrumentator`
- **app-go** — `/metrics` via `prometheus/client_golang` (HTTP instrumentado)
- **nginx-exporter** — métricas derivadas do **`stub_status`** do Nginx (o exporter consulta o Nginx; em seguida o Prometheus consulta o exporter em `:9113`)

No Grafana, pasta **To Brasil** (provisionada):

- **TO Brasil — Aplicações** — taxas e latência HTTP (Python e Go)
- **TO Brasil — Nginx** — requests, conexões e estados (`reading` / `writing` / `waiting`)

## Arquitetura e análise

### Fluxo de requisição

1. Cliente faz request para `http://localhost/python/time` ou `http://localhost/go/time`
2. Nginx verifica se há resposta em cache para a URI
3. **MISS**: encaminha para o upstream, armazena a resposta, retorna ao cliente
4. **HIT**: retorna a resposta cacheada diretamente, sem tocar no upstream
5. Header `X-Cache-Status` indica o resultado (MISS, HIT, STALE, etc.)

### Fluxo de atualização de cada componente

#### Código das aplicações (app-python / app-go)

1. Alterar o código-fonte na respectiva pasta
2. `docker compose build app-python` ou `docker compose build app-go`
3. `docker compose up -d app-python` ou `docker compose up -d app-go`
4. Nginx continua rodando; o cache expira naturalmente e passa a servir a versão nova

**Melhoria**: usar CI/CD para build automático de imagens e deploy via rolling update (ex: Kubernetes Deployment com strategy RollingUpdate).

#### Configuração do Nginx (cache, rotas)

1. Editar `nginx/nginx.conf`
2. `docker compose restart nginx`

**Melhoria**: validar a config antes de aplicar (`nginx -t`) — pode ser feito com um healthcheck customizado.

#### Prometheus / Grafana

1. Editar os respectivos arquivos de configuração
2. `docker compose restart prometheus` ou `docker compose restart grafana`

**Melhoria**: usar `/-/reload` do Prometheus para recarregar a config sem restart.

### Pontos de melhoria identificados

#### Segurança
- **TLS/HTTPS**: adicionar certificados (Let's Encrypt ou cert-manager no Kubernetes) ao Nginx.
- **Rate limiting**: configurar no Nginx para proteger contra abuso.
- **Network isolation**: criar redes Docker separadas (frontend/backend) para que apenas o Nginx fique exposto.
- **Grafana**: trocar a senha padrão e configurar autenticação via OAuth/SSO.

#### Resiliência e Escalabilidade
- **Health checks**: adicionar endpoints `/health` nas aplicações e configurar healthchecks no Docker Compose.
- **Réplicas**: usar `deploy.replicas` no Compose ou migrar para Kubernetes para escalar horizontalmente.
- **Load balancing**: com múltiplas réplicas, Nginx já suporta upstream balancing (round-robin, least_conn, etc.).

#### Observabilidade
- **Logging centralizado**: adicionar Loki ou ELK stack para agregar logs de todos os serviços.
- **Tracing distribuído**: integrar Jaeger ou Tempo para rastrear requests entre Nginx e as aplicações.
- **Dashboards Grafana**: provisionar dashboards prontos para métricas HTTP e cache hit rate.
- **Alerting**: configurar alertas no Prometheus (Alertmanager) para métricas críticas (ex: error rate > 5%, cache hit rate baixo).

#### CI/CD
- **Pipeline**: configurar GitHub Actions (ou GitLab CI) para lint, testes, build de imagens e push para registry.
- **Versionamento de imagens**: usar tags semânticas (v1.0.0) ao invés de `latest`.
- **Infrastructure as Code**: para produção, usar Terraform/Pulumi para provisionar a infra e Kubernetes manifests (ou Helm charts) para deploy.

#### Cache
- **Cache keys**: ajustar as chaves de cache se houver query parameters ou headers que variem.
- **Cache externo**: para cenários mais complexos, considerar Redis/Varnish como camada de cache dedicada.
- **Purge**: implementar mecanismo de invalidação manual de cache (nginx cache purge module).

## Estrutura do projeto

```
.
├── app-python/ # FastAPI
│   ├── main.py
│   ├── requirements.txt
│   └── Dockerfile
├── app-go/                     # net/http
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── nginx/
│   └── nginx.conf              # Proxy reverso + cache
├── prometheus/
│   └── prometheus.yml
├── grafana/
│   └── provisioning/
│       ├── dashboards/
│       │   ├── dashboards.yml
│       │   └── json/
│       │       ├── apps.json
│       │       └── nginx.json
│       └── datasources/
│           └── prometheus.yml
├── architecture-diagram/       # Diagrama “as code” + PNG
│   ├── architecture.py
│   ├── requirements.txt
│   ├── README.md
│   └── to_brasil_arquitetura.png
├── docker-compose.yml
└── README.md
```
