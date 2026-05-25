# Transfer

Send folders between computers on your network. Files are compressed and encrypted with a password you choose before being transferred.

## Requirements

- Go 1.21+
- Both computers on the same network

## Usage

Clone the repo and run:

```bash
git clone https://github.com/kikikian/transfer
cd transfer
go run main.go
```

Open `http://localhost:3030` in your browser.

### Sending a folder

1. Open the sender computer's browser at `http://localhost:3030`
2. Click **Send**
3. Enter the folder path, a password, and the receiver's IP address (e.g. `192.168.1.45:8080`)
4. Click Send

### Receiving a folder

1. Open the receiver computer's browser at `http://localhost:3030`
2. Click **Receive**
3. Enter the output folder path and the same password used by the sender
4. Click Receive — it will wait for the incoming transfer

### Finding your IP address

On Linux/Mac:
```bash
ip addr
```

On Windows:
```bash
ipconfig
```

Look for something like `192.168.1.x`.

## Tickets

current tickets:
- [x] Backend (logic in [[transfer backend notes]])
	- [x] create compressing and decompressing logic
	- [x] create function to listen on port
	- [x] create function to dial port
		- [x] also create `handleConnection func` on server side
	- [x] logic to stream data through port
	- [x] create main function to handle everything
    - [ ] logging
- [x] UI
	- [x] Website
		- [x] Form requests
		- [x] progress bar
        - [ ] add visualization of logs (graph)
	