FROM golang:1.24-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@v1.61.7

COPY go.mod ./
# Copia go.sum se existir, senão ignora para não quebrar o build inicial
COPY go.* ./
RUN go mod download || true

COPY . .

EXPOSE 8080

# Executa go mod tidy para garantir dependências e inicia o air para hot reload
CMD ["sh", "-c", "go mod tidy && air -c .air.toml"]
