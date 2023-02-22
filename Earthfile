VERSION 0.6

FROM golang:1.20

all-bench:
    BUILD +fmt-bench
    BUILD +lint-bench
    BUILD +vet-bench
    BUILD +test-bench

all-interface:
    BUILD +fmt-interface
    BUILD +lint-interface
    BUILD +vet-interface
    BUILD +test-interface

all-generic:
    BUILD +fmt-generic
    BUILD +lint-generic
    BUILD +vet-generic
    BUILD +test-generic

all:
    BUILD +all-bench
    BUILD +all-interface
    BUILD +all-generic

ci-bench:
    ARG --required BUILD_NUMBER
    BUILD +fmt-bench
    BUILD +lint-bench
    BUILD +vet-bench
    COPY --dir +test-bench/files ./test-results/bench
    SAVE ARTIFACT ./test-results/bench AS LOCAL test-results/bench

    BUILD +publish-coverage-bench --BUILD_NUMBER=$BUILD_NUMBER

ci-interface:
    ARG --required BUILD_NUMBER
    BUILD +fmt-interface
    BUILD +lint-interface
    BUILD +vet-interface
    COPY --dir +test-interface/files ./test-results/interface
    SAVE ARTIFACT ./test-results/interface AS LOCAL test-results/interface
    BUILD +publish-coverage-interface --BUILD_NUMBER=$BUILD_NUMBER

ci-generic:
    ARG --required BUILD_NUMBER
    BUILD +fmt-generic
    BUILD +lint-generic
    BUILD +vet-generic
    COPY --dir +test-generic/files ./test-results/generic
    SAVE ARTIFACT ./test-results/generic AS LOCAL test-results/generic
    BUILD +publish-coverage-generic --BUILD_NUMBER=$BUILD_NUMBER

ci:
    ARG --required BUILD_NUMBER
    BUILD +ci-bench --BUILD_NUMBER=$BUILD_NUMBER
    BUILD +ci-interface --BUILD_NUMBER=$BUILD_NUMBER
    BUILD +ci-generic --BUILD_NUMBER=$BUILD_NUMBER

go-mod-bench:
    WORKDIR /bench
    RUN git config --global url."git@github.com:".insteadOf "https://github.com/"
    RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts
    COPY go.mod go.sum .
    RUN --ssh go mod download

go-mod-interface:
    WORKDIR /app
    RUN git config --global url."git@github.com:".insteadOf "https://github.com/"
    RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts
    COPY go.mod go.sum .
    RUN --ssh go mod download

go-mod-generic:
    WORKDIR /generic
    RUN git config --global url."git@github.com:".insteadOf "https://github.com/"
    RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts
    COPY go.mod go.sum .
    RUN --ssh go mod download

go-mod:
    BUILD +go-mod-bench
    BUILD +go-mod-interface
    BUILD +go-mod-generic

fmt-bench:
    COPY --dir ./bench /bench
    WORKDIR /bench
    RUN find . -type d -path "./vendor" -prune -o -name "*.go" -exec gofmt -d -e {} \; | tee /tmp/gofmt.out
    RUN bash -c 'if [[ -s /tmp/gofmt.out ]]; then exit 1; fi'

fmt-interface:
    COPY --dir . /app
    RUN rm -rf generic
    RUN rm -rf bench
    RUN find . -type d -path "./vendor" -prune -o -name "*.go" -exec gofmt -d -e {} \; | tee /tmp/gofmt.out
    RUN bash -c 'if [[ -s /tmp/gofmt.out ]]; then exit 1; fi'

fmt-generic:
    COPY --dir ./generic /generic
    WORKDIR /generic
    RUN find . -type d -path "./vendor" -prune -o -name "*.go" -exec gofmt -d -e {} \; | tee /tmp/gofmt.out
    RUN bash -c 'if [[ -s /tmp/gofmt.out ]]; then exit 1; fi'

fmt:
    BUILD +fmt-bench
    BUILD +fmt-interface
    BUILD +fmt-generic

vendor-bench:
    FROM +go-mod-bench
    WORKDIR /app
    COPY --dir . .
    WORKDIR /app/bench
    RUN --ssh go mod vendor
    SAVE ARTIFACT . files

vendor-interface:
    FROM +go-mod-interface
    WORKDIR /app
    COPY --dir . .
    RUN rm -rf generic
    RUN rm -rf bench
    RUN --ssh go mod vendor
    SAVE ARTIFACT . files

vendor-generic:
    FROM +go-mod-generic
    WORKDIR /app
    COPY --dir .git .
    COPY ./generic ./generic
    WORKDIR /app/generic
    RUN --ssh go mod vendor
    SAVE ARTIFACT . files

vendor:
    BUILD +vendor-bench
    BUILD +vendor-interface
    BUILD +vendor-generic

lint-bench:
    FROM +vendor-bench
    COPY +golangci-lint/go/bin/golangci-lint /go/bin/golangci-lint
    RUN golangci-lint run

lint-interface:
    FROM +vendor-interface
    COPY +golangci-lint/go/bin/golangci-lint /go/bin/golangci-lint
    RUN golangci-lint run

lint-generic:
    FROM +vendor-generic
    COPY +golangci-lint/go/bin/golangci-lint /go/bin/golangci-lint
    RUN golangci-lint run

lint:
    BUILD +lint-bench
    BUILD +lint-interface
    BUILD +lint-generic

vet-bench:
    FROM +vendor-bench
    RUN go vet ./...

vet-interface:
    FROM +vendor-interface
    RUN go vet ./...

vet-generic:
    FROM +vendor-generic
    RUN go vet ./...

vet:
    BUILD +vet-bench
    BUILD +vet-interface
    BUILD +vet-generic

test-bench:
    FROM +vendor-bench
    COPY +junit-report/go/bin/go-junit-report /go/bin/go-junit-report
    RUN go version
    RUN mkdir -p test-results
    # To both see the output in the console AND convert into junit-style results
    # to send to the plug-in, we need to run the tests, writing to a file, then
    # send that file to go-junit-report
    RUN 2>&1 go test -race -v ./... -cover -coverprofile=test-results/cover.out | tee test-results/go-test-bench.out
    RUN cat test-results/go-test-bench.out | $GOPATH/bin/go-junit-report > test-results/go-test-bench-report.xml
    SAVE ARTIFACT test-results files

test-interface:
    FROM +vendor-interface
    COPY +junit-report/go/bin/go-junit-report /go/bin/go-junit-report
    RUN go version
    RUN mkdir -p test-results
    # To both see the output in the console AND convert into junit-style results
    # to send to the plug-in, we need to run the tests, writing to a file, then
    # send that file to go-junit-report
    RUN 2>&1 go test -race -v ./... -cover -coverprofile=test-results/cover.out | tee test-results/go-test-interface.out
    RUN cat test-results/go-test-interface.out | $GOPATH/bin/go-junit-report > test-results/go-test-interface-report.xml
    SAVE ARTIFACT test-results files

test-generic:
    FROM +vendor-generic
    COPY +junit-report/go/bin/go-junit-report /go/bin/go-junit-report
    RUN go version
    RUN mkdir -p test-results
    # To both see the output in the console AND convert into junit-style results
    # to send to the plug-in, we need to run the tests, writing to a file, then
    # send that file to go-junit-report
    RUN 2>&1 go test -race -v ./... -cover -coverprofile=test-results/cover.out | tee test-results/go-test-generic.out
    RUN cat test-results/go-test-generic.out | $GOPATH/bin/go-junit-report > test-results/go-test-generic-report.xml
    SAVE ARTIFACT test-results files

test:
    COPY --dir +test-bench/files ./test-results/bench
    COPY --dir +test-interface/files ./test-results/interface
    COPY --dir +test-generic/files ./test-results/generic
    SAVE ARTIFACT ./test-results AS LOCAL test-results

publish-coverage-interface:
    ARG --required BUILD_NUMBER
    FROM +test-interface
    COPY +goveralls/go/bin/goveralls /go/bin/goveralls
    RUN --no-cache --secret COVERALLS_TOKEN=+secrets/COVERALLS_TOKEN \
        goveralls \
        -jobnumber="$BUILD_NUMBER" \
        -flagname=interface \
        -service=buildkite \
        -coverprofile=test-results/cover.out

publish-coverage-bench:
    ARG --required BUILD_NUMBER
    FROM +test-bench
    COPY +goveralls/go/bin/goveralls /go/bin/goveralls
    RUN --no-cache --secret COVERALLS_TOKEN=+secrets/COVERALLS_TOKEN \
        goveralls \
        -jobnumber="$BUILD_NUMBER" \
        -flagname=bench \
        -service=buildkite \
        -coverprofile=test-results/cover.out

publish-coverage-generic:
    ARG --required BUILD_NUMBER
    FROM +test-generic
    COPY +goveralls/go/bin/goveralls /go/bin/goveralls
    RUN --no-cache --secret COVERALLS_TOKEN=+secrets/COVERALLS_TOKEN \
        goveralls \
        -jobnumber="$BUILD_NUMBER" \
        -flagname=generic \
        -service=buildkite \
        -coverprofile=test-results/cover.out

# These are tools that are used in the targets above
goveralls:
    RUN echo Installing goveralls
    RUN go install github.com/mattn/goveralls@latest
    SAVE ARTIFACT /go/bin/goveralls /go/bin/goveralls

golangci-lint:
    RUN echo Installing golangci-lint...
    # see https://golangci-lint.run/usage/install/#other-ci
    RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /go/bin v1.51.2
    SAVE ARTIFACT /go/bin/golangci-lint /go/bin/golangci-lint

junit-report:
    RUN go install github.com/jstemmer/go-junit-report@latest
    SAVE ARTIFACT /go/bin/go-junit-report /go/bin/go-junit-report
