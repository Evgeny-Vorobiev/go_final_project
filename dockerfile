# Этап сборки: компиляция бинарного файла под Linux
FROM golang:latest AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Отключаем CGO и явно указываем целевую ОС для статической сборки
RUN CGO_ENABLED=0 GOOS=linux go build -o scheduler .

# Финальный образ: минимальная среда для запуска
FROM ubuntu:latest
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /build/scheduler .
COPY web ./web

EXPOSE 7540

# Переменные окружения по умолчанию
ENV TODO_PORT=7540
ENV TODO_DBFILE=/data/scheduler.db

ENTRYPOINT ["./scheduler"]
