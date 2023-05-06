FROM alpine:latest
WORKDIR /nudge
COPY nudge .
COPY config.yml .
COPY nudge.private-key.pem .
COPY static /nudge/static
CMD ["./nudge"]
EXPOSE 9000