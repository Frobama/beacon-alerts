# beacon-alerts

Servicio de **alertas** de Beacon. Recibe webhooks (Alertmanager), los normaliza, deduplica por fingerprint y reporta a Incidentes.

| | |
|---|---|
| Lenguaje | Go |
| Base de datos | TimescaleDB |
| Expone | gRPC (`AlertService`) y HTTP para webhooks |
| Consume | gRPC de `IncidentService` |

## Contratos

Los stubs se generan desde [`beacon-contracts`](https://github.com/TU_ORG/beacon-contracts):

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

## Estado

Fase 0: solo generación de contratos. El código llega en la Fase 1/2.
