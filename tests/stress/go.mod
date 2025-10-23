module github.com/luciancaetano/kephasnet/tests/stress

go 1.25.1

require (
	github.com/gorilla/websocket v1.5.3
	github.com/luciancaetano/kephasnet v0.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/time v0.14.0 // indirect
)

replace github.com/luciancaetano/kephasnet => ../..
