# Go Web Development - Project `lenslocked`

## Project-Specific Note

### Initialize

1. Setup Go module. (Follow `Maintain Go Module`)
2. Create executable with a command `go install .`
3. [PROJECT NAME]
i.e. lenslocked


## General Note

### Run server
$ go run main.go

### Maintain Go Module
1. initiate go module (go.mod)
$ go mod init [MODULE DIRECTORY] 
i.e. go mod init github.com/uhdang/lenslocked

2. Occasionally tidy things up, remove unused libraries and keep your mod file tidy:
$ go mod tidy

### Dynamic reloading
- Using `modd` - https://github.com/cortesi/modd
- `modd.conf` for configuration
- Run server with dynamic reloading (+test)
  $ modd

### Adding package
a. Installing `gorilla/mux`
$ go get -u github.com/gorilla/mux
