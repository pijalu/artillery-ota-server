# Artillery OTA Server

A reimplementation of the Artillery OTA (Over-The-Air) server. The ARM version can be run on the printer itself to provide local firmware updates.

## Overview

This is a custom OTA firmware server implementation that mimics the official Artillery firmware update service. It allows you to host your own firmware updates for Artillery printers, providing a local alternative to the official update mechanism.

## Features

- Compatible with Artillery printers' firmware update mechanism
- Support for multiple firmware types and customer configurations
- Ability to embed firmware files directly in the executable
- Cross-platform support (Linux, macOS, Windows)
- ARM64 support for running on the printer itself

## Installation

### Pre-built Binaries

Download the appropriate binary for your platform from the [releases page](https://github.com/pijalu/artillery-ota-server/releases).

### Building from Source

1. Install Go 1.25 or later
2. Clone this repository:

```bash
git clone https://github.com/pijalu/artillery-ota-server.git
cd artillery-ota-server
```

3. Build the project:

```bash
make build-all
```

Binaries will be placed in the `bin/` directory.

## Configuration

The server configuration is defined in `config.json`. You can customize which firmware files are served by modifying this file. The configuration supports:

- Multiple customer types (e.g., Yuntu_m1)
- Different firmware types
- Embedded files (files compiled into the executable)
- File system files (external files)

## Running on the Printer

To run the OTA server directly on your Artillery printer (ARM version):

1. Add the following line to `/etc/hosts` on your printer:
   ```
   127.0.0.1       studio.ota.artillery3d.com
   ```

2. Copy the `ota-server-linux-arm64` binary to your printer and place it in a suitable location (e.g., `/usr/local/bin/`)

3. Configure the server to start automatically by adding it to `/etc/rc.local`:

   ```bash
   #!/bin/sh -e
   # Add this line before the exit 0

   # Start the OTA server in the background (default is localhost:9190)
   /usr/local/bin/ota-server-linux-arm64 &

   exit 0
   ```

4. Make sure `/etc/rc.local` is executable:
   ```bash
   chmod +x /etc/rc.local
   ```

5. Optionally, create a systemd service for better management:

   Create `/etc/systemd/system/artillery-ota.service`:
   ```ini
   [Unit]
   Description=Artillery OTA Server
   After=network.target

   [Service]
   Type=simple
   User=root
   ExecStart=/usr/local/bin/ota-server-linux-arm64
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```

   Then enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable artillery-ota.service
   sudo systemctl start artillery-ota.service
   ```

## Usage

Run the server with default settings (binds to localhost:9190):
```bash
./ota-server
```

Run the server with custom bind address and port (if needed):
```bash
./ota-server -bind 0.0.0.0 -port 9190
```

Enable request tracing for debugging:
```bash
./ota-server -trace
```

## API Endpoints

- `/home/downloadnewest?customerType=X&firmwareType=Y` - Returns firmware metadata in the expected format
- `/download/{filename}` - Downloads the actual firmware file
- `/upload/firmware/{filename}` - Alternative endpoint for firmware downloads

## Development

### Embedding Files

The project uses Go's embed functionality to include firmware files directly in the built executable. To regenerate the embedded files:

```bash
go generate
```

Or use the Makefile:
```bash
make generate
```

### Building for Different Platforms

Use the Makefile targets:

```bash
# Build all platforms
make build-all

# Build specific platform
make build-linux-amd64
make build-linux-arm64
make build-darwin-arm64
make build-windows-amd64
```

### Testing

Run the tests:
```bash
make test
```

## Security Considerations

- The server binds to localhost by default for security
- File path validation prevents directory traversal attacks
- Only files specified in the configuration can be downloaded

## License

MIT License - See the LICENSE file for details.