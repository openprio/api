FROM golang as builder
RUN mkdir /go/src/openprio_api
WORKDIR /go/src/openprio_api
COPY . .

RUN go get
RUN CGO_ENABLED=0 go build -o /go/bin/openprio_api

FROM alpine
COPY --from=builder /go/bin/openprio_api /app/openprio_api
WORKDIR /app
CMD ["/app/openprio_api"]
