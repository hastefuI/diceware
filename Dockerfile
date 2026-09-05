FROM golang:1.27.1-trixie AS build
WORKDIR /src

ARG VERSION=dev

COPY go.mod go.sum ./
RUN go mod download

COPY diceware.go ./
COPY cmd ./cmd

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/diceware ./cmd/diceware

FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /app

COPY --from=build /out/diceware /app/diceware
COPY wordlists/wordlist-basque-diceware.txt /app/wordlists/wordlist-basque-diceware.txt

ENTRYPOINT ["/app/diceware"]
