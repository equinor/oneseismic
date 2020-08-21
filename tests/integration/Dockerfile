FROM python:buster
WORKDIR /code
COPY /tests/integration/requirements.txt .
RUN pip install -r requirements.txt
COPY /core/ingestion/upload/requirements-dev.txt .
RUN pip install -r requirements-dev.txt

COPY /core/ingestion .
RUN pip install ./upload
RUN pip install ./scan

COPY /client client
RUN pip install ./client

COPY /tests/integration/integration_tests.py .
COPY /tests/integration/data/small.sgy .

COPY /tests/ssl/az.pem  /etc/ssl/certs/
ENV REQUESTS_CA_BUNDLE=/etc/ssl/certs/az.pem
ENTRYPOINT ["pytest","-v","integration_tests.py"]
