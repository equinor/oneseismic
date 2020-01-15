FROM python:3.6
RUN pip install pipenv
WORKDIR /src
COPY Pipfile* ./
RUN pipenv install --dev --system --skip-lock
ADD . .

RUN black --check .
RUN cd scan && flake8 && cd ..
RUN cd upload && flake8 && cd ..
RUN cd blobio && flake8 && cd ..
RUN cd tests && flake8 && cd ..
RUN isort -c

RUN pip install ./scan
RUN pip install ./upload
RUN pip install ./blobio

RUN pytest upload scan blobio

ENTRYPOINT [ "pytest", "tests" ]
