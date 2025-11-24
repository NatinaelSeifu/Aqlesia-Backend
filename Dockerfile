FROM golang:1.24-alpine AS builder
WORKDIR /
ADD . .
RUN apk add --no-cache git
RUN go build -o bin/aqlesia cmd/main.go

FROM alpine:latest
RUN apk add --no-cache curl

WORKDIR /

COPY --from=builder /bin/aqlesia .
COPY --from=builder /config/config.yaml /config/config.yaml
COPY --from=builder /internal/constants/query/schemas /internal/constants/query/schemas
COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /zoneinfo.zip

ENV ZONEINFO=/zoneinfo.zip

EXPOSE 8000

HEALTHCHECK --timeout=3s CMD curl --fail http://localhost:8000/v1/auth/health || exit 1

ENTRYPOINT [ "./aqlesia" ]