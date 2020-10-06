FROM python:buster

WORKDIR /python
COPY /python .
# Prerelease versions of azure-storage-blob might use versions of x-ms-version
# that Azurite do not support yet.
# For now, just trust ms to not release azure-storage-blob versions until Azurite
# supports the same x-ms-version value.
# By using pip to install packages, prerelease versions will not be installed.
RUN pip install -r requirements-dev.txt
RUN python3 setup.py install
RUN python3 setup.py test

WORKDIR /tests
COPY /tests/integration_tests.py .
COPY /tests/requirements.txt .
RUN pip install -r requirements.txt

COPY /tests/ssl/az.pem  /etc/ssl/certs/
ENV REQUESTS_CA_BUNDLE=/etc/ssl/certs/az.pem
ENTRYPOINT ["pytest","-v","integration_tests.py"]
