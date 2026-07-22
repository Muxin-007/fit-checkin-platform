# Tasks For FIT CHECKIN Boot

## build

> build platform

```sh
REV_COUNT=$(git rev-list --count HEAD)
REVISION=$(git rev-parse HEAD)
BRANCH=$(git branch --show-current)
COMPILE_TIME=$(date +"%Y-%m-%d %H:%M:%S")
COMMIT_DATE=$(git show -s --format=%cd --date=short HEAD)
GO_VERSION=$(go version)
MODULE_NAME=$(cat MODULE_NAME)
MODULE_VERSION=$(cat VERSION)

CGO_ENABLED=0 go build -gcflags "-l -l" -ldflags "-X main.moduleName=$MODULE_NAME -X main.moduleVersion=$MODULE_VERSION -X main.branch=$BRANCH -X main.revision=$REVISION -X 'main.goVersion=$GO_VERSION' -X 'main.compileTime=$COMPILE_TIME'" -tags=go_json -o bin/$MODULE_NAME main.go
```

## package [suffix]

> package platform docker

```sh
set -ex
DOCKER_REGISTRY=harbor.seike.cn/fit-checkin
PACKAGE_TIME=$(date +"%Y%m%d")
if [ -n "$suffix" ]; then
    IMAGE_NAME=platform-${suffix}
else
    IMAGE_NAME=platform
fi

mask build

docker build -t $DOCKER_REGISTRY/$IMAGE_NAME:$PACKAGE_TIME -f ./Dockerfile .

docker tag $DOCKER_REGISTRY/$IMAGE_NAME:$PACKAGE_TIME $DOCKER_REGISTRY/$IMAGE_NAME:latest
```

## clean

> clean

```sh
rm -rf bin
```