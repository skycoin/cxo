CONTRIBUTING
============

Code quality restrictions:

- all `.go` source files should be formated using `gofmt`
- follow Golang naming convention: CamelCase, `Value()/SetValue()`,
  `IsExist()`, etc

And recommendations:

- all `.go` source files should be formated using `gofmt` _with `-s` flag_
- length of a code line should not exceed 80 characters
- tests, examples and benchmarks should follow Golang naming convention
- `go vet` (`go vet ./...`)
- [`gocyclo`](https://github.com/fzipp/gocyclo) (`gocyclo -over 15 .`)

And more recommendations:

- [golint](https://github.com/golang/lint)
- [ineffassign](https://github.com/gordonklaus/ineffassign)
  (`ls ./**/*.go | xargs -L 1 ineffassign` or `ineffassign file.go`)
- [misspell](https://github.com/client9/misspell) (`misspell .`)
- you feature should have test case, that proof that it works
