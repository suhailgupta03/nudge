FROM alpine:latest
WORKDIR /nudge
COPY nudge .
COPY config.yml .
COPY nudgetest.2023-04-14.private-key.pem .
CMD ["./nudge"]
EXPOSE 9000