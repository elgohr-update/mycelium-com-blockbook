FROM golang:1.14-stretch

RUN apt-get update && apt-get install -y \
    build-essential git pkg-config libzmq3-dev libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev
RUN git clone --depth 1 --single-branch -b v6.14.6 https://github.com/facebook/rocksdb.git /rocksdb; \
    cd /rocksdb; CFLAGS="-fPIC -mno-avx -mno-avx2" CXXFLAGS=-fPIC make release

WORKDIR /blockbook
ADD . .
RUN CGO_CFLAGS="-I/rocksdb/include" CGO_LDFLAGS="-L/rocksdb -lrocksdb -lstdc++ -lm -lz -ldl -lbz2 -lsnappy -llz4" CFLAGS="-mno-avx -mno-avx2" go build
