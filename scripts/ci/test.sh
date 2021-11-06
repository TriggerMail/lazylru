go env
go get -t -v ./...
go get -v -u github.com/jstemmer/go-junit-report
go vet ./...

# format
find . -type d -path "./vendor" -prune -o -name "*.go" -exec gofmt -d {} \; | tee /tmp/gofmt.out
if [[ -s /tmp/gofmt.out ]]; then exit 1; fi

# run tests
export TEST_RESULTS=test-results
mkdir -p $TEST_RESULTS
trap "$GOPATH/bin/go-junit-report <${TEST_RESULTS}/go-test.out > ${TEST_RESULTS}/go-test-report.xml" EXIT
go test -v ./... | tee ${TEST_RESULTS}/go-test.out