module github.com/shiestapoi/whatsmeow

go 1.24

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/rs/zerolog v1.33.0
	go.mau.fi/libsignal v0.1.2
	go.mau.fi/util v0.8.5
	golang.org/x/crypto v0.36.0
	golang.org/x/net v0.37.0
	google.golang.org/protobuf v1.36.5
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.31.0 // indirect
)

// This prevents go mod tidy from updating protobuf to v1.36.5
replace google.golang.org/protobuf => google.golang.org/protobuf v1.31.0

replace github.com/shiestapoi/whatsmeow => github.com/shiestapoi/whatsmeow v0.0.0-20250315104305-fecd891dc006
