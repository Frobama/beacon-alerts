# beacon-alerts

Servicio de **alertas** de Beacon. Recibe webhooks (Alertmanager), los normaliza, deduplica por fingerprint y reporta a Incidentes.

| | |
|---|---|
| Lenguaje | Go |
| Base de datos | TimescaleDB |
| Expone | gRPC (`AlertService`) y HTTP para webhooks |
| Consume | gRPC de `IncidentService` |

## Contratos

Los stubs se generan desde [`beacon-contracts`](https://github.com/Frobama/beacon-contracts):

```bash
make proto                          # usa la versión fijada en el Makefile
make proto CONTRACTS_TAG=v0.2.0     # probar otra versión
```

El código generado queda en `gen/` y no se versiona.

## Entorno local

La base de datos corre desde [`beacon-infra`](https://github.com/Frobama/beacon-infra) (`make up`):

```text
postgres://beacon:beacon@localhost:5433/alerts
```

## Salud

| Endpoint | Qué comprueba |
|---|---|
| `GET /livez` | que el proceso vive; no toca la base |
| `GET /healthz` | que la base responde |
| gRPC `grpc.health.v1.Health/Check` (9091) | estado del servidor gRPC |

## Migraciones

Por defecto el servicio migra al arrancar (`MIGRATE_ON_START=true`), cómodo para desarrollo local.
En Kubernetes las migraciones corren en un Job de Helm **antes** del despliegue:

```text
/app/beacon-alerts migrate
```

Con varias réplicas, `MIGRATE_ON_START` debe ser `false`: si no, todas compiten por migrar.

## Variables de entorno

| Variable | Obligatoria | Default | Para qué |
|---|---|---|---|
| `HTTP_ADDR` | sí | — | webhook y probes, por ejemplo `:8082` |
| `GRPC_ADDR` | sí | — | `AlertService`, por ejemplo `:9091` |
| `DATABASE_URL` | sí | — | TimescaleDB |
| `INCIDENTS_GRPC_ADDR` | sí (servidor) | — | `ReportAlert`; el subcomando `migrate` no lo exige |
| `WEBHOOK_TOKEN` | en Kubernetes | vacío | token `Authorization: Bearer`; vacío = sin auth (solo local) |
| `MIGRATE_ON_START` | no | `true` | migrar al arrancar; `false` en el cluster |
| `DEDUP_WINDOW` | no | `30m` | ventana de deduplicación |
| `REPORT_TIMEOUT` | no | `5s` | timeout de `ReportAlert` |
| `RECONCILE_INTERVAL` | no | `30s` | intervalo del reconciliador |
| `RECONCILE_MARGIN` | no | `10s` | margen para no competir con el webhook |

## Cobertura

```bash
make coverage          # falla bajo 60%
make coverage THRESHOLD=101  # comprobación de que el umbral falla
```

El script mide `./internal/...` y `./cmd/...`, sin el código generado. Los tests de
`store` requieren `TEST_DATABASE_URL`; sin esa variable se saltan (el CI levanta un
Postgres de servicio).

## Imagen

Multi-etapa con distroless `static-debian12:nonroot`, usuario `65532:65532`, versión y
commit incrustados con `-ldflags`. No incluye shell ni toolchain de Go.

## Pipeline

`.github/workflows/ci.yml` genera los stubs, corre `go vet` y `make coverage` contra un
Postgres de servicio, construye la imagen, la escanea con Trivy y publica en GHCR solo en
`master` y tags `v*` (tags `sha-…`, `master` y `vX.Y.Z`, nunca `latest`). El aviso a
`beacon-infra` usa el secreto `INFRA_DISPATCH_TOKEN` y se omite si no existe.
