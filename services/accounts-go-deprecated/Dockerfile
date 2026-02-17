FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -o server ./cmd

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/server .
EXPOSE 8081
CMD ["./server"]
