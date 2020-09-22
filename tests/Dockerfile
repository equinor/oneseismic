FROM python:buster
WORKDIR /code
COPY /tests/requirements.txt .
RUN pip install -r requirements.txt

COPY /python python
RUN pip install python/

COPY /tests/integration_tests.py .
COPY /tests/data data

COPY /tests/ssl/az.pem  /etc/ssl/certs/
ENV REQUESTS_CA_BUNDLE=/etc/ssl/certs/az.pem
ENTRYPOINT ["pytest","-v","integration_tests.py"]
