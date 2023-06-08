FROM alpine:latest

RUN apk add --no-cache tzdata

WORKDIR /nudge
COPY nudge .
COPY config.yml .
COPY nudge.private-key.pem .
COPY static /nudge/static
CMD ["./nudge"]
EXPOSE 9000