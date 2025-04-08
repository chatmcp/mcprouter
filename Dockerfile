FROM golang:1.23.8-alpine3.20 AS build

WORKDIR /build
COPY ./ ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o mcprouter -ldflags '-w -s' main.go

FROM alpine

# install kubectl
RUN apk add --no-cache curl && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/ && \
    rm -rf /var/cache/apk/*

WORKDIR /data

COPY ./.env.example.toml ./.env.toml
COPY --from=build /build/mcprouter ./mcprouter

EXPOSE 8025 8027

RUN chmod +x mcprouter

ENTRYPOINT ["./mcprouter"]

CMD ["proxy"]
