FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ranges ./cmd/ranges

FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/ranges /usr/local/bin/ranges
EXPOSE 8083
ENTRYPOINT ["ranges"]
