# whatsmeow
[![Go Reference](https://pkg.go.dev/badge/github.com/shiestapoi/whatsmeow.svg)](https://pkg.go.dev/github.com/shiestapoi/whatsmeow)

whatsmeow is a Go library for the WhatsApp web multidevice API.

## Discussion
Matrix room: [#whatsmeow:maunium.net](https://matrix.to/#/#whatsmeow:maunium.net)

For questions about the WhatsApp protocol (like how to send a specific type of
message), you can also use the [WhatsApp protocol Q&A] section on GitHub
discussions.

[WhatsApp protocol Q&A]: https://github.com/tulir/whatsmeow/discussions/categories/whatsapp-protocol-q-a

## Usage
The [godoc](https://pkg.go.dev/github.com/shiestapoi/whatsmeow) includes docs for all methods and event types.
There's also a [simple example](https://pkg.go.dev/github.com/shiestapoi/whatsmeow#example-package) at the top.

## Installation

To use this library in your project:

```bash
# Add the module to your project
go get github.com/shiestapoi/whatsmeow

# Ensure you have the correct protobuf version
go get google.golang.org/protobuf@v1.31.0

# Run go mod tidy to clean up dependencies
go mod tidy
```

If the module disappears after running `go mod tidy`, ensure that:
1. You're not importing the module in a circular way
2. Your imports reference the correct package path
3. Your go.mod doesn't contain a self-reference to the module

## Features
Most core features are already present:

* Sending messages to private chats and groups (both text and media)
* Receiving all messages
* Managing groups and receiving group change events
* Joining via invite messages, using and creating invite links
* Sending and receiving typing notifications
* Sending and receiving delivery and read receipts
* Reading and writing app state (contact list, chat pin/mute status, etc)
* Sending and handling retry receipts if message decryption fails
* Sending status messages (experimental, may not work for large contact lists)

Things that are not yet implemented:

* Sending broadcast list messages (this is not supported on WhatsApp web either)
* Calls

## Troubleshooting

### Protobuf Errors

If you encounter errors like:
```
panic: runtime error: slice bounds out of range [-1:]
```

It may be due to protobuf version compatibility issues. You can fix this by:

1. Run the fix script:
   ```bash
   go run fix_protobuf_errors.go
   ```

2. Or upgrade to the latest version:
   ```bash
   go get -u github.com/shiestapoi/whatsmeow@latest
   ```

3. You can also downgrade protobuf in your project:
   ```bash
   go get google.golang.org/protobuf@v1.31.0
   ```

### Module Dependency Issues

If you see your module dependency disappearing after `go mod tidy`:

1. Check for circular dependencies in your go.mod file
2. Make sure you're not referring to the module itself as a dependency
3. Use the replace directive if needed:
   ```
   replace github.com/original/module => github.com/shiestapoi/whatsmeow v0.0.0-20250315104305-fecd891dc006
   ```
