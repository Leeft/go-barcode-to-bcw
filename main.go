package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/gousb"
)

// HIDUsageMap translates USB HID Scan Codes to ASCII runes.
var HIDUsageMap = map[byte][2]rune{
	0x04: {'a', 'A'}, 0x05: {'b', 'B'}, 0x06: {'c', 'C'}, 0x07: {'d', 'D'},
	0x08: {'e', 'E'}, 0x09: {'f', 'F'}, 0x0a: {'g', 'G'}, 0x0b: {'h', 'H'},
	0x0c: {'i', 'I'}, 0x0d: {'j', 'J'}, 0x0e: {'k', 'K'}, 0x0f: {'l', 'L'},
	0x10: {'m', 'M'}, 0x11: {'n', 'N'}, 0x12: {'o', 'O'}, 0x13: {'p', 'P'},
	0x14: {'q', 'Q'}, 0x15: {'r', 'R'}, 0x16: {'s', 'S'}, 0x17: {'t', 'T'},
	0x18: {'u', 'U'}, 0x19: {'v', 'V'}, 0x1a: {'w', 'W'}, 0x1b: {'x', 'X'},
	0x1c: {'y', 'Y'}, 0x1d: {'z', 'Z'},
	0x1e: {'1', '!'}, 0x1f: {'2', '@'}, 0x20: {'3', '#'}, 0x21: {'4', '$'},
	0x22: {'5', '%'}, 0x23: {'6', '^'}, 0x24: {'7', '&'}, 0x25: {'8', '*'},
	0x26: {'9', '('}, 0x27: {'0', ')'},
	0x28: {'\n', '\n'}, 0x2a: {'\b', '\b'}, 0x2c: {' ', ' '},
	0x2d: {'-', '_'}, 0x2e: {'=', '+'}, 0x2f: {'[', '{'}, 0x30: {']', '}'},
	0x31: {'\\', '|'}, 0x33: {';', ':'}, 0x34: {'\'', '"'}, 0x35: {'`', '~'},
	0x36: {',', '<'}, 0x37: {'.', '>'}, 0x38: {'/', '?'},
}

func parseID(input string) uint16 {
	input = strings.TrimPrefix(input, "0x")
	val, err := strconv.ParseUint(input, 16, 16)
	if err != nil {
		valDec, errDec := strconv.ParseUint(input, 10, 16)
		if errDec != nil {
			return 0
		}
		return uint16(valDec)
	}
	return uint16(val)
}

// Payload defines the JSON structure for the API
type Payload struct {
	Code      string    `json:"barcode"`
	Timestamp time.Time `json:"timestamp"`
}

func sendToAPI(apiURL, apiKey, code string) {
	data := Payload{
		Code:      code,
		Timestamp: time.Now(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error encoding JSON: %v", err)
		return
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("BBUDDY-API-KEY", apiKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("API request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Successfully sent barcode: %s (Status: %d)\n", code, resp.StatusCode)
	} else {
		log.Printf("API returned error: %d", resp.StatusCode)
	}
}

func main() {
	// 1. Define Flags and Env Vars
	flagVID := flag.String("vid", os.Getenv("SCANNER_VID"), "USB Vendor ID")
	flagPID := flag.String("pid", os.Getenv("SCANNER_PID"), "USB Product ID")
	flagURL := flag.String("api-url", os.Getenv("API_URL"), "API endpoint URL")
	flagKey := flag.String("api-key", os.Getenv("API_KEY"), "API Authorization Key")
	flag.Parse()

	if *flagVID == "" || *flagPID == "" || *flagURL == "" || *flagKey == "" {
		fmt.Println("Usage: program -vid=0xXXXX -pid=0xXXXX -api-url=https://... -api-key=xyz")
		os.Exit(1)
	}

	vid := gousb.ID(parseID(*flagVID))
	pid := gousb.ID(parseID(*flagPID))

	// 2. Initialize USB
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(vid, pid)
	if err != nil || dev == nil {
		log.Fatalf("Could not open device: %v", err)
	}
	defer dev.Close()

	// Iterate through configurations
	// for _, cfgDesc := range dev.Desc.Configs {
	// 	fmt.Printf("Config %d:\n", cfgDesc.Number)
	// 	for _, intfDesc := range cfgDesc.Interfaces {
	// 		fmt.Printf("  Interface %d:\n", intfDesc.Number)
	// 		for _, altDesc := range intfDesc.AltSettings {
	// 			fmt.Printf("    Alternate Setting %d:\n", altDesc.Number)
	// 			for _, epDesc := range altDesc.Endpoints {
	// 				fmt.Printf("      Endpoint %d %s (%s)\n",
	// 					epDesc.Number, epDesc.Direction, epDesc.TransferType)
	// 			}
	// 		}
	// 	}
	// }

	dev.SetAutoDetach(true)

	intf, done, err := dev.DefaultInterface()
	if err != nil {
		log.Fatalf("Failed to claim interface: %v", err)
	}
	defer done()

	epIn, err := intf.InEndpoint(1)
	if err != nil {
		epIn, err = intf.InEndpoint(129)
		if err != nil {
			log.Fatalf("Could not open endpoint: %v", err)
		}
	}

	fmt.Printf("Scanner active. Sending codes to: %s\n", *flagURL)

	var barcode strings.Builder
	var lastKey byte

	for {
		buf := make([]byte, 8)
		_, err := epIn.Read(buf)
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		isShift := (buf[0]&0x02 != 0) || (buf[0]&0x20 != 0)
		keyCode := buf[2]

		if keyCode == 0 {
			lastKey = 0
			continue
		}
		if keyCode == lastKey {
			continue
		}
		lastKey = keyCode

		val, ok := HIDUsageMap[keyCode]
		if !ok {
			continue
		}

		char := val[0]
		if isShift {
			char = val[1]
		}

		if char == '\n' || char == '\r' {
			scannedCode := barcode.String()
			if len(scannedCode) > 0 {
				fmt.Printf("Local Scan: %s\n", scannedCode)
				// Send to API in the background or synchronously
				go sendToAPI(*flagURL, *flagKey, scannedCode)
				barcode.Reset()
			}
			continue
		}
		barcode.WriteRune(char)
	}
}
