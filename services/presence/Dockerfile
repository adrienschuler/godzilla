FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd

FROM scratch
WORKDIR /app
COPY --from=build /app/server .
EXPOSE 50051
CMD ["./server"]
