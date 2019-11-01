FROM golang:1.13.2 as bd
WORKDIR /appB
ADD . .
RUN go build -ldflags="-w -s" -a -o /app .

FROM alpine
RUN apk --update add ca-certificates
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY --from=bd /app /app/app
WORKDIR /app
CMD ./app
