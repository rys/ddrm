#!/usr/local/bin/zsh

DDRM_GIT_SHA=$(git rev-parse --short HEAD)
DDRM_GIT_TAG=$(git describe --tags 2>/dev/null)
DDRM_GIT_DATE=$(git log -1 --format=%cd --date=short)

[[ -z ${DDRM_GIT_TAG} ]] && DDRM_GIT_TAG=dev

DDRM_OS=$(uname)
DDRM_ARCH=$(uname -m)

DDRM_BIN_PATH=./bin/${DDRM_GIT_TAG}
DDRM_DIST_PATH=./dist

DDRM_CLIENT_BIN="ddrm-client-${DDRM_GIT_TAG}-${DDRM_OS:l}-${DDRM_ARCH:l}"

DDRM_BUILD_USER=${USER}

[[ -f ${DDRM_BIN_PATH}/${DDRM_CLIENT_BIN} ]] && rm -f ${DDRM_BIN_PATH}/${DDRM_CLIENT_BIN} && echo "Removed existing ${DDRM_BIN_PATH}/${DDRM_CLIENT_BIN}"

LDFLAGS="-X main.BuildVersion=${DDRM_GIT_TAG} -X main.BuildDate=${DDRM_GIT_DATE} -X main.GitRev=${DDRM_GIT_SHA} -X main.BuildUser=${DDRM_BUILD_USER}"

[[ $DDRM_GIT_TAG != "dev" ]] && echo "Not packaging a dev build, stripping symbol table and debug info" && LDFLAGS="${LDFLAGS} -s -w"

if [ -v "SKIP_BUILD" ]; then
    echo "Skipping build"
else
    GOOS=linux   GOARCH=amd64 go build -ldflags ${LDFLAGS} -tags client -o ${DDRM_BIN_PATH}/ddrm-client-${DDRM_GIT_TAG}-linux-amd64
    GOOS=darwin  GOARCH=amd64 go build -ldflags ${LDFLAGS} -tags client -o ${DDRM_BIN_PATH}/ddrm-client-${DDRM_GIT_TAG}-darwin-x86_64
    GOOS=freebsd GOARCH=amd64 go build -ldflags ${LDFLAGS} -tags client -o ${DDRM_BIN_PATH}/ddrm-client-${DDRM_GIT_TAG}-freebsd-amd64
    GOOS=linux   GOARCH=arm64 go build -ldflags ${LDFLAGS} -tags client -o ${DDRM_BIN_PATH}/ddrm-client-${DDRM_GIT_TAG}-linux-arm64
    GOOS=darwin  GOARCH=arm64 go build -ldflags ${LDFLAGS} -tags client -o ${DDRM_BIN_PATH}/ddrm-client-${DDRM_GIT_TAG}-darwin-arm64
    GOOS=freebsd GOARCH=arm64 go build -ldflags ${LDFLAGS} -tags client -o ${DDRM_BIN_PATH}/ddrm-client-${DDRM_GIT_TAG}-freebsd-arm64
fi

[[ ! -f ${DDRM_BIN_PATH}/${DDRM_CLIENT_BIN} ]] && echo "Didn't build client binary for ${DDRM_OS:l}-${DDRM_ARCH:l}: ${DDRM_CLIENT_BIN}"
[[ -f ${DDRM_BIN_PATH}/${DDRM_CLIENT_BIN} ]]   && echo "Built client binary for ${DDRM_OS:l}-${DDRM_ARCH:l}: ${DDRM_BIN_PATH}/${DDRM_CLIENT_BIN}"

if [ -v "CREATE_PACKAGE" ]; then
    for i (linux-amd64 darwin-x86_64 freebsd-amd64 linux-arm64 darwin-arm64 freebsd-arm64) do
        rel=ddrm-client-${DDRM_GIT_TAG}-${i}
        echo "Packaging ${rel}"
        out=${DDRM_DIST_PATH}/${rel}
        mkdir -p ${out}/doc
        mkdir -p ${out}/bin
        cp ./doc/ddrm.conf-example                  ${out}/doc
        cp ./doc/ddrm-records.conf-example          ${out}/doc
        cp ./README.md                              ${out}
        cp ./LICENSE                                ${out}
        cp ${DDRM_BIN_PATH}/${rel}                  ${out}/bin
        tar czf ${out}.tar.gz -C ${DDRM_DIST_PATH}  ${rel}
        rm -rf ${out}
    done
fi