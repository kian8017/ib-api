FROM golang as builder

WORKDIR /usr/src/ib

COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY ./*.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -v -o ib ./...

FROM alpine
WORKDIR /root/
# Add ripgrep
RUN apk add ripgrep

# The IndexBrain API expects the names to be inside /names
RUN mkdir /names

COPY --from=builder /usr/src/ib/ib ./

CMD ["./ib"]
