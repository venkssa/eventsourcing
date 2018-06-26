FROM alpine:3.5

COPY bin/eventsourcing /usr/local/eventsourcing

CMD [ "/usr/local/eventsourcing" ]
