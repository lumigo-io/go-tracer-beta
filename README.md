
![CircleCI](https://circleci.com/gh/lumigo-io/go-tracer-beta/tree/master.svg?style=svg&circle-token=421fefe82bcad1c17c4116f154e25e32ebc90f2c)
[![Go Report Card](https://goreportcard.com/badge/github.com/lumigo-io/go-tracer-beta)](https://goreportcard.com/report/github.com/lumigo-io/go-tracer-beta)
[![GoDoc](https://godoc.org/github.com/lumigo-io/go-tracer-beta?status.svg)](https://godoc.org/github.com/lumigo-io/go-tracer-beta)

# go-tracer (BETA)

This is lumigo/go-tracer-beta, Lumigo's Golang agent for distributed tracing and performance monitoring.

## Installation

`go-tracer-beta` can be installed like any other Go library through `go get`:

```console
$ go get github.com/lumigo-io/go-tracer-beta
```

Or, if you are already using
[Go Modules](https://github.com/golang/go/wiki/Modules), you may specify a
version number as well:

```console
$ go get github.com/lumigo-io/go-tracer-beta@master
```

## Usage

You need a lumigo token which you can find under the `Project Settings` and `Tracing` tab in lumigo portal. Then you need just to wrap your Lambda:

```go
type MyEvent struct {
  Name string `json:"name"`
}

func HandleRequest(ctx context.Context, name MyEvent) (string, error) {
  return fmt.Sprintf("Hello %s!", name.Name ), nil
}

func main() {
	wrappedHandler := lumigotracer.WrapHandler(HandleRequest, &lumigotracer.Config{
		Token:       "<your-token>",
	})
	lambda.Start(wrappedHandler)
}
```

## Contributing
Contributions to this project are welcome from all! Below are a couple pointers on how to prepare your machine, as well as some information on testing.

### Required Tools:
- go v1.16 and later
- make

If you want to deploy the example lambda for real testing you need: 
- terraform 0.14.5

### Lint

Linting the codebase:
```
make lint
```

### Test suite

Run the test suite:
```
make test
```

### Check styles

Runs go vet and lint in parallel

```
make checks
```

### Deploy example

Deploys in AWS a lambda function wrapped by tracer and prints tracing in logs (stdout):

```
export AWS_PROFILE=<your-profile>
make deploy-example
```

After you finished testing just destroy the AWS infrastructure resources for Lambda:

```
export AWS_PROFILE=<your-profile>
make destroy-example
```