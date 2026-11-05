FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /web ./cmd/web

FROM alpine:3.20
RUN adduser -D -H app
WORKDIR /app
COPY --from=build /web /app/web
COPY templates ./templates
COPY static ./static
COPY content ./content
USER app
EXPOSE 8080
CMD ["/app/web"]
