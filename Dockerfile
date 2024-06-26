FROM golang:1.22.3
ENV TZ=America/Los_Angeles

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o 4k2rss

ENTRYPOINT ["./4k2rss"]
