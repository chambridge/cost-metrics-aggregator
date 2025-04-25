FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

RUN microdnf install -y tar gzip && microdnf clean all

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o /app/server ./cmd/server

EXPOSE 8080

CMD ["/app/server"]
