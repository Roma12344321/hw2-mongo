FROM golang:1.27 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /bin/lab ./cmd/lab

FROM alpine:3.22
COPY --from=build /bin/lab /bin/lab

ENTRYPOINT ["/bin/lab"]
