FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod .
COPY main.go .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /app-go .

FROM alpine:3.21
RUN adduser -D -u 1000 appuser
COPY --from=build /app-go /app-go
USER appuser
EXPOSE 8080
CMD ["/app-go"]
