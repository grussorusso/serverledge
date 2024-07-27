# Conventions for tests in this project

- All test files should end with _test
- All test files should be in this folder
- All test functions should start with Test*
- All errors must be checked to be nil with utils.AssertNil() or utils.AssertNilMsg()
- If a test requires serverledge to be running, use:

```go
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
```
- All test utility functions should be into util.go and should be private (to this folder)
- To run all tests, use:

```sh
make test
```
- To run only short tests, use:

```sh
make test -short
```
- You can run a test individually with
 
```sh
    go test -v -run MyTestFunction
```