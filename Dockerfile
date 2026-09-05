FROM golang:1.27-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY diceware.go ./
COPY cmd ./cmd

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/diceware ./cmd/diceware

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/diceware /app/diceware
COPY wordlists/wordlist-basque-diceware.txt /app/wordlists/wordlist-basque-diceware.txt

ENTRYPOINT ["/app/diceware"]
