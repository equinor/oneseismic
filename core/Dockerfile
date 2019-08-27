FROM ubuntu:19.10 as dependencies

RUN apt-get update
RUN apt-get -y install curl

RUN curl https://packages.microsoft.com/config/ubuntu/18.04/packages-microsoft-prod.deb --output packages-microsoft-prod.deb 
RUN dpkg -i packages-microsoft-prod.deb
RUN apt-get update

RUN apt-get -y install \
    cmake \
    clang \
    libclang-dev \
    make \
    gcc \
    g++ \
    libc-dev \
    linux-headers-generic \
    git \
    python3 \
    python3-pip \
    python3-dev \
    blobfuse \
    gettext


ENV PYTHONPATH /usr/local/lib/python3.7/site-packages
RUN python3 -m  pip install scikit-build pybind11 grpcio grpcio-tools protobuf

RUN git clone https://github.com/statoil/segyio
WORKDIR /segyio
RUN cmake . -DEXPERIMENTAL=ON -DBUILD_PYTHON=OFF -DBUILD_TESTING=OFF -DBUILD_SHARED_LIBS=OFF
RUN make
RUN make install



FROM  dependencies as builder
RUN mkdir -p /build
COPY . /src
WORKDIR /build
RUN cmake ../src -DBUILD_SHARED_LIBS=off -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr/local
RUN make 
RUN make install

COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh
RUN mkdir -p /mnt/ramdisk
RUN mount -v -t tmpfs -o size=1g tmpfs /mnt/ramdisk
RUN mkdir /mnt/ramdisk/blobfusetmp

FROM  builder as deploy

# RUN adduser --disabled-password appuser
# USER appuser
# WORKDIR /home/appuser

EXPOSE 5000

ENTRYPOINT ["entrypoint.sh"]
CMD ["core-server","help"]
