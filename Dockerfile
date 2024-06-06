FROM golang:alpine3.20 as build

WORKDIR /app

COPY . ./

RUN go mod download && go mod verify

RUN CGO_ENABLED=0 GOOS=linux go build -o /vaultdump

FROM golang:alpine3.20

WORKDIR /vault

COPY --from=build /vaultdump /vaultdump

RUN apk add --no-cache postgresql-client

CMD ["/vaultdump"]
