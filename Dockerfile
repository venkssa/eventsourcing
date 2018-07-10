FROM alpine:3.7

COPY bin/eventsourcing /usr/local/eventsourcing

CMD [ "/usr/local/eventsourcing" ]
