FROM python:3.10-alpine3.14

RUN mkdir -p /app
COPY bin/executor /

COPY images/python310/entrypoint.py /

CMD /executor
