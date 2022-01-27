FROM python:3.8-buster AS cppbuild
RUN apt-get update && apt-get install --no-install-recommends -y \
    build-essential \
    cmake \
    libfmt-dev \
    libmsgpack-dev
COPY ./core ./core
RUN cmake \
      -S core \
      -B /app/cppbuild \
      -DCMAKE_BUILD_TYPE=Release \
      -DBUILD_TESTING=ON \
      -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
      -DCMAKE_CXX_FLAGS=-DFMT_HEADER_ONLY=1
RUN cmake --build /app/cppbuild --config Release


FROM cppbuild AS pythonbuild
RUN apt-get update && apt-get install --no-install-recommends -y \
    build-essential \
    cmake \
    libfmt-dev \
    libmsgpack-dev
COPY ./python /python
WORKDIR /python
RUN python -m pip install --user --upgrade pip
RUN python -m pip install --user -r requirements-dev.txt
RUN python setup.py build_ext --inplace build  -- -Doneseismic_DIR="/app/cppbuild"
RUN python setup.py test


FROM pythonbuild AS pythontest
COPY ./tests /tests
RUN python -m pip install -r /tests/requirements.txt
RUN python -m venv /pyenv/upload_with
RUN python -m venv /pyenv/fetch_with

RUN /pyenv/upload_with/bin/python -m pip install --upgrade pip
RUN /pyenv/upload_with/bin/pip install -r /python/requirements-dev.txt
RUN /pyenv/upload_with/bin/pip install -r /tests/requirements.txt

RUN /pyenv/fetch_with/bin/python -m pip install --upgrade pip
RUN /pyenv/fetch_with/bin/pip install -r /python/requirements-dev.txt
RUN /pyenv/fetch_with/bin/pip install -r /tests/requirements.txt

ARG UPLOAD_WITH_CLIENT_VERSION
ENV UPLOAD_WITH_CLIENT_VERSION=$UPLOAD_WITH_CLIENT_VERSION
RUN if test -z "$UPLOAD_WITH_CLIENT_VERSION"; \
    then echo /python > /pyenv/upload_with/lib/python3.8/site-packages/oneseismic.pth; \
    else /pyenv/upload_with/bin/pip install oneseismic=="$UPLOAD_WITH_CLIENT_VERSION"; \
    fi

ARG FETCH_WITH_CLIENT_VERSION
ENV FETCH_WITH_CLIENT_VERSION=$FETCH_WITH_CLIENT_VERSION
RUN if test -z "$FETCH_WITH_CLIENT_VERSION"; \
    then echo /python > /pyenv/fetch_with/lib/python3.8/site-packages/oneseismic.pth; \
    else /pyenv/fetch_with/bin/pip install oneseismic=="$FETCH_WITH_CLIENT_VERSION"; \
    fi

ENV UPLOAD_WITH_PYTHON /pyenv/upload_with/bin/python
WORKDIR /tests


FROM pythontest AS localtest
ENTRYPOINT ["/pyenv/fetch_with/bin/python","-m", "pytest", "-s", "-m", "local"]


FROM pythontest AS versiontest
ENTRYPOINT ["/pyenv/fetch_with/bin/python","-m", "pytest", "-s", "-m", "version"]
