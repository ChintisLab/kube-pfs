FROM alpine:3.21

ARG COMPONENT=unknown
LABEL org.opencontainers.image.title="kube-pfs ${COMPONENT} smoke image"

RUN adduser -D -u 10001 appuser
USER appuser

CMD ["sh", "-c", "echo kube-pfs ${COMPONENT} smoke container started"]
