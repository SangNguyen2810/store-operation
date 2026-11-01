FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/stores ./cmd/stores

FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/stores /usr/local/bin/stores
EXPOSE 8081
ENTRYPOINT ["stores"]
