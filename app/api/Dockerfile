# Build context is the repository root, because the API uses the shared module:
#   docker build -f app/api/Dockerfile .
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY shared/go.mod shared/go.sum shared/
COPY app/api/go.mod app/api/go.sum app/api/
RUN cd app/api && go mod download
COPY shared shared
COPY app/api app/api
RUN cd app/api && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/api /usr/local/bin/api
EXPOSE 8081
ENTRYPOINT ["api"]
