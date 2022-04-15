FROM golang AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache libc6-compat
RUN apk update && apk add --no-cache libc6-compat  gcompat

COPY --from=builder /app/main /app/main

CMD ["/app/main"]
