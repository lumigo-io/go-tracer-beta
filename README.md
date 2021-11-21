# go-tracer

## Required Tools:
- go v1.16 and later
- make

## Development

For development you need: 
- terraform 0.14.5

### Lint

Linting the codebase:
```
make lint
```

### Test

Run the tests:
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