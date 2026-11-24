-- +goose Up
CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE alerts (
    id                UUID PRIMARY KEY,
    source            TEXT        NOT NULL,
    fingerprint       TEXT        NOT NULL,
    title             TEXT        NOT NULL,
    service           TEXT        NOT NULL,
    severity          SMALLINT    NOT NULL,
    labels            JSONB       NOT NULL DEFAULT '{}'::jsonb,
    raw_payload       JSONB       NOT NULL,
    occurrences       INTEGER     NOT NULL DEFAULT 1,
    incident_id       UUID,
    first_received_at TIMESTAMPTZ NOT NULL,
    last_received_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX alerts_fingerprint_idx ON alerts (fingerprint, first_received_at DESC);
CREATE INDEX alerts_service_idx     ON alerts (service, last_received_at DESC);
CREATE INDEX alerts_incident_idx    ON alerts (incident_id) WHERE incident_id IS NOT NULL;
CREATE INDEX alerts_unassigned_idx  ON alerts (last_received_at DESC) WHERE incident_id IS NULL;

CREATE TABLE alert_events (
    id          UUID        NOT NULL DEFAULT gen_random_uuid(),
    alert_id    UUID        NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    service     TEXT        NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    payload     JSONB       NOT NULL,
    PRIMARY KEY (id, received_at)
);

SELECT create_hypertable('alert_events', 'received_at', migrate_data => true);

CREATE INDEX alert_events_service_idx ON alert_events (service, received_at DESC);
CREATE INDEX alert_events_alert_idx   ON alert_events (alert_id, received_at DESC);

-- +goose Down
DROP TABLE alert_events;
DROP TABLE alerts;
