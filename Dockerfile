FROM golang:1.25-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /out/wow1 .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/wow1 /wow1
EXPOSE 8080
ENTRYPOINT ["/wow1"]
CMD ["server"]
