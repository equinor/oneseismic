FROM python:3.6
RUN pip install pipenv
WORKDIR /src
COPY Pipfile* ./
RUN pipenv install --dev --system --skip-lock
ADD . .

RUN pip install ./scan
RUN pip install ./upload
RUN pip install ./blobio

RUN pytest upload scan

RUN flake8

ENTRYPOINT [ "pytest", "tests" ]
