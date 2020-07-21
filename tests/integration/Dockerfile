FROM python:3.6-alpine
RUN apk add --no-cache curl pkgconfig openssl-dev libffi-dev musl-dev make gcc
WORKDIR /code
COPY /tests/integration/requirements.txt .
RUN pip install -r requirements.txt

COPY /tests/integration/integration_tests.py .
COPY /tests/ssl/az.pem  /etc/ssl/certs/
ENV REQUESTS_CA_BUNDLE=/etc/ssl/certs/az.pem
ENTRYPOINT ["pytest","-v","integration_tests.py"]
