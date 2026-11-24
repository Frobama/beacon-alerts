#!/usr/bin/env bash
# Falla si la cobertura total de ./internal/... y ./cmd/... baja del umbral.
# RACE_FLAG="" (vacío) permite correrlo en Windows sin CGO; sin definir, usa -race.
set -euo pipefail

threshold="${1:-60}"
packages="./internal/... ./cmd/..."

if [ -n "${RACE_FLAG:-}" ]; then
	race="$RACE_FLAG"
else
	race=""
fi

# shellcheck disable=SC2086
go test $packages $race -covermode=atomic -coverprofile=coverage.out

total="$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$3); print $3}')"
printf 'cobertura total: %s%% (mínimo %s%%)\n' "$total" "$threshold"

awk -v total="$total" -v min="$threshold" 'BEGIN { if (total + 0 < min + 0) exit 1 }'
