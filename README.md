# go-barcode-to-bcw

A lightweight Go application that reads barcodes from USB HID devices (barcode scanners) and forwards them to the [Grocy Barcode Wizard](https://github.com/Leeft/grocy-barcode-wizard) API in real-time.

## Features

- **USB HID Scanner Support**: Reads barcode data directly from USB-connected barcode scanners
- **Real-time Processing**: Scans are immediately sent to the API endpoint
- **Simple Configuration**: Environment variables or command-line flags for setup
- **Docker Ready**: Includes Dockerfile for containerized deployment
- **Low Resource Footprint**: Minimal dependencies, runs on any Linux system

## Requirements

### For Direct Installation

- **Go 1.26+** (if building from source)
- **libusb-1.0**: USB library for device communication
- A USB barcode scanner with HID support
- Network connectivity to the Grocy Barcode Wizard API

### For Docker

- **Docker** and Docker Compose
- USB device passthrough configured on the host

## Installation

### Option 1: Docker (Recommended)

1. Build the Docker image:

   ```bash
   docker build -t go-barcode-to-bcw:latest .
   ```

2. Run the container with the appropriate environment variables (see [Configuration](#configuration) section):

   ```bash
   docker run -it --privileged \
     -v /dev/bus/usb:/dev/bus/usb \
     -e SCANNER_VID="0x0581" \
     -e SCANNER_PID="0x0115" \
     -e API_URL="http://grocy-barcode-wizard:3000/api/action/scan" \
     -e API_KEY="your-api-key" \
     go-barcode-to-bcw:latest
   ```

### Option 2: Docker Compose

Include in your docker-compose.yml:

```yaml
services:
  barcode-scanner:
    image: go-barcode-to-bcw:latest
    container_name: barcode-scanner
    privileged: true
    stdin_open: true
    tty: true
    environment:
      SCANNER_VID: "0x0581"
      SCANNER_PID: "0x0115"
      API_URL: "http://bcw:3000/api/action/scan"
      API_KEY: "your-api-key"
    volumes:
      - /dev/bus/usb:/dev/bus/usb
      - /dev/usb:/dev/usb
      - /etc/timezone:/etc/timezone:ro
      - /etc/localtime:/etc/localtime:ro
    restart: unless-stopped
```

### Option 3: Direct Installation

1. Install libusb:

   ```bash
   # Ubuntu/Debian
   sudo apt-get install libusb-1.0-0-dev

   # macOS
   brew install libusb
   ```

2. Build the application:

   ```bash
   go build -o go-barcode-to-bcw
   ```

3. Run with environment variables:

   ```bash
   export SCANNER_VID="0x0581"
   export SCANNER_PID="0x0115"
   export API_URL="http://localhost:3000/api/action/scan"
   export API_KEY="your-api-key"
   
   ./go-barcode-to-bcw
   ```

   Or use command-line flags:

   ```bash
   ./go-barcode-to-bcw \
     -vid=0x0581 \
     -pid=0x0115 \
     -api-url="http://localhost:3000/api/action/scan" \
     -api-key="your-api-key"
   ```

## Configuration

### Required Parameters

- **SCANNER_VID**: USB Vendor ID of your barcode scanner (hexadecimal format)
- **SCANNER_PID**: USB Product ID of your barcode scanner (hexadecimal format)
- **API_URL**: URL endpoint of the Grocy Barcode Wizard API
- **API_KEY**: API key for authentication with the Grocy Barcode Wizard server

### Finding Your Scanner's VID and PID

#### On Linux

1. **Install lsusb** (if not already installed):
   ```bash
   sudo apt-get install usbutils
   ```

2. **Connect your barcode scanner and run**:
   ```bash
   lsusb
   ```

3. **Look for your scanner in the output**:
   ```
   Bus 001 Device 005: ID 0581:0115 MYWAY-ELECTRONICS LTD. bar code scanner
                          ^^^^  ^^^^
                          VID   PID
   ```

4. **Use the values** (e.g., `SCANNER_VID="0x0581"` and `SCANNER_PID="0x0115"`)

#### On macOS

1. Run:
   ```bash
   system_profiler SPUSBDataType | grep -A5 "barcode\|scanner"
   ```

2. Look for `Vendor ID` and `Product ID` in the output

#### On Windows

1. Open **Device Manager**
2. Locate your barcode scanner under **USB devices**
3. Right-click → **Properties** → **Details** tab
4. Select **Hardware IDs** from the dropdown
5. You'll see entries like `VID_0581&PID_0115` - extract the values

### Barcode Format

The application supports standard HID keyboard input from barcode scanners. It captures keystrokes until a newline character (Enter key) is detected, then sends the complete barcode string to the API.

## Usage

Once configured, simply:

1. **Connect your USB barcode scanner** to the host machine
2. **Start the application** (via Docker or direct command)
3. **Scan barcodes** - each scan will be sent to the API endpoint with a timestamp

The application logs each scan to stdout:
```
Local Scan: 5901234123457
Successfully sent barcode: 5901234123457 (Status: 200)
```

## Docker Compose Example

A complete docker-compose example with Grocy Barcode Wizard:

```yaml
version: '3.8'

services:
  bcw:
    image: grocy-barcode-wizard:latest
    ports:
      - "3000:3000"
    environment:
      GROCY_URL: https://grocy.example.com/
      GROCY_API_URL: https://grocy.example.com/api/
      GROCY_API_KEY: LAIxSjB6UlxtJSM8a6jabNzwhCms10Bx2X      
    volumes:
      - bcw_data:/app/data

  barcode-scanner:
    image: go-barcode-to-bcw:latest
    container_name: barcode-scanner
    privileged: true
    stdin_open: true 
    tty: true
    depends_on:
      - bcw
    environment:
      SCANNER_VID: "0x0581"
      SCANNER_PID: "0x0115"
      API_URL: "http://bcw:3000/api/action/scan"
      API_KEY: "your-secure-api-key"
    volumes:
      - /dev/bus/usb:/dev/bus/usb
      - /dev/usb:/dev/usb
    restart: unless-stopped

volumes:
  bcw_data:
```

## Troubleshooting

### "Could not open device" Error

- **Issue**: Scanner not detected
- **Solution**: 
  - Verify the scanner is connected: `lsusb | grep <SCANNER_NAME>`
  - Check that VID and PID are correct
  - Ensure the device has USB permissions (may need `sudo`)
  - Try using `sudo` to run the application if using direct installation

### "API request failed" Error

- **Issue**: Cannot reach the API endpoint
- **Solution**:
  - Verify the API_URL is correct and reachable: `curl http://api-url/`
  - Check network connectivity between scanner host and API server
  - Verify the API_KEY is correct
  - Check that the API server is running

### Scanner Works but Barcodes Not Being Sent

- **Issue**: Barcodes read locally but not received by API
- **Solution**:
  - Check application logs for errors
  - Verify the API endpoint URL is correct
  - Test API connectivity: `curl -X POST http://api-url -H "Authorization: Bearer KEY"`
  - Check that the scanner barcode suffix is set to send Enter/newline after each scan

### Permission Issues

- **Docker**: Use `privileged: true` in docker-compose.yml
- **Direct**: Run with `sudo` or add your user to the `plugdev` group:
  ```bash
  sudo usermod -a -G plugdev $USER
  ```

## License

MIT License - See LICENSE file for details.
