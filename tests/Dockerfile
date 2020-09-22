FROM python:buster

WORKDIR /python
COPY /python .
RUN python3 setup.py install
RUN python3 setup.py test

WORKDIR /tests
COPY /tests/integration_tests.py .
COPY /tests/data data
COPY /tests/requirements.txt .
RUN pip install -r requirements.txt

COPY /tests/ssl/az.pem  /etc/ssl/certs/
ENV REQUESTS_CA_BUNDLE=/etc/ssl/certs/az.pem
ENTRYPOINT ["pytest","-v","integration_tests.py"]
