# Arquitetura e Análise

## Diagrama

```
    ┌─────────┐
    │ Cliente  │
    └────┬─────┘
         │ :80
    ┌────▼──────────────────────────────────────────┐
    │                  Nginx                         │
    │  Reverse Proxy + Cache                         │
    │                                                │
    │  /python/*  ─► proxy_pass app-python:8000      │
    │                proxy_cache_valid 10s            │
    │                                                │
    │  /go/*      ─► proxy_pass app-go:8080          │
    │                proxy_cache_valid 60s            │
    │                                                │
    │  /stub_status ─► métricas internas do Nginx    │
    └────┬──────────────────┬───────────────────────┘
         │                  │
    ┌────▼─────────┐  ┌────▼──────────┐
    │ app-python   │  │   app-go      │
    │ FastAPI      │  │   net/http    │
    │ :8000        │  │   :8080       │
    │              │  │               │
    │ GET /        │  │ GET /         │
    │ GET /time    │  │ GET /time     │
    │ GET /metrics │  │ GET /metrics  │
    └────┬─────────┘  └────┬──────────┘
         │                  │
         └───────┬──────────┘
                 │
    ┌────────────▼──────────────┐
    │  nginx-exporter :9113     │  pull stub_status no Nginx
    └────────────┬──────────────┘
                 │
    ┌────────────▼──────────────┐
    │    Prometheus :9090        │
    │  scrape: app-python,      │
    │          app-go,           │
    │          nginx-exporter    │
    └────────────┬──────────────┘
                 │
    ┌────────────▼──────────────┐
    │      Grafana :3000        │
    │  Datasource: Prometheus   │
    └───────────────────────────┘
```

## Fluxo de requisição

1. Cliente faz request para `http://localhost/python/time` ou `http://localhost/go/time`
2. Nginx verifica se há resposta em cache para a URI
3. **MISS**: encaminha para o upstream, armazena a resposta, retorna ao cliente
4. **HIT**: retorna a resposta cacheada diretamente, sem tocar no upstream
5. Header `X-Cache-Status` indica o resultado (MISS, HIT, STALE, etc.)

## Fluxo de atualização de cada componente

### Código das aplicações (app-python / app-go)

1. Alterar o código-fonte na respectiva pasta
2. `docker compose build app-python` ou `docker compose build app-go`
3. `docker compose up -d app-python` ou `docker compose up -d app-go`
4. Nginx continua rodando; o cache expira naturalmente e passa a servir a versão nova

**Melhoria**: usar CI/CD para build automático de imagens e deploy via rolling update (ex: Kubernetes Deployment com strategy RollingUpdate).

### Configuração do Nginx (cache, rotas)

1. Editar `nginx/nginx.conf`
2. `docker compose restart nginx`

**Melhoria**: validar a config antes de aplicar (`nginx -t`) — pode ser feito com um healthcheck customizado.

### Prometheus / Grafana

1. Editar os respectivos arquivos de configuração
2. `docker compose restart prometheus` ou `docker compose restart grafana`

**Melhoria**: usar `/-/reload` do Prometheus para recarregar a config sem restart.

## Pontos de melhoria identificados

### Segurança
- **TLS/HTTPS**: adicionar certificados (Let's Encrypt ou cert-manager no Kubernetes) ao Nginx.
- **Rate limiting**: configurar no Nginx para proteger contra abuso.
- **Network isolation**: criar redes Docker separadas (frontend/backend) para que apenas o Nginx fique exposto.
- **Grafana**: trocar a senha padrão e configurar autenticação via OAuth/SSO.

### Resiliência e Escalabilidade
- **Health checks**: adicionar endpoints `/health` nas aplicações e configurar healthchecks no Docker Compose.
- **Réplicas**: usar `deploy.replicas` no Compose ou migrar para Kubernetes para escalar horizontalmente.
- **Load balancing**: com múltiplas réplicas, Nginx já suporta upstream balancing (round-robin, least_conn, etc.).

### Observabilidade
- **Logging centralizado**: adicionar Loki ou ELK stack para agregar logs de todos os serviços.
- **Tracing distribuído**: integrar Jaeger ou Tempo para rastrear requests entre Nginx e as aplicações.
- **Dashboards Grafana**: provisionar dashboards prontos para métricas HTTP e cache hit rate.
- **Alerting**: configurar alertas no Prometheus (Alertmanager) para métricas críticas (ex: error rate > 5%, cache hit rate baixo).

### CI/CD
- **Pipeline**: configurar GitHub Actions (ou GitLab CI) para lint, testes, build de imagens e push para registry.
- **Versionamento de imagens**: usar tags semânticas (v1.0.0) ao invés de `latest`.
- **Infrastructure as Code**: para produção, usar Terraform/Pulumi para provisionar a infra e Kubernetes manifests (ou Helm charts) para deploy.

### Cache
- **Cache keys**: ajustar as chaves de cache se houver query parameters ou headers que variem.
- **Cache externo**: para cenários mais complexos, considerar Redis/Varnish como camada de cache dedicada.
- **Purge**: implementar mecanismo de invalidação manual de cache (nginx cache purge module).
