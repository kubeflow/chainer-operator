FROM golang:1.10.2-alpine3.7 AS build

# Install tools required to build the project.
# We need to run `docker build --no-cache .` to update those dependencies.
RUN apk add --no-cache git bash
RUN go get github.com/golang/dep/cmd/dep

# Gopkg.toml and Gopkg.lock lists project dependencies.
# These layers are only re-built when Gopkg files are updated.
COPY Gopkg.lock Gopkg.toml /go/src/github.com/kubeflow/chainer-operator/
WORKDIR /go/src/github.com/kubeflow/chainer-operator/

# Install library dependencies.
RUN dep ensure -vendor-only

# Copy all project and build it.
# This layer is rebuilt when ever a file has changed in the project directory.
COPY . /go/src/github.com/kubeflow/chainer-operator/
RUN ./hack/verify-codegen.sh && go test github.com/kubeflow/chainer-operator/...
RUN go build -o /bin/chainer-operator github.com/kubeflow/chainer-operator/cmd/chainer-operator

FROM alpine:3.7
COPY --from=build /bin/chainer-operator /bin/chainer-operator
ENTRYPOINT ["/bin/chainer-operator"]
CMD ["--help"]
