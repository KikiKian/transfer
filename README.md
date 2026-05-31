# Transfer

Send folders securely between computers on your network. Files are automatically compressed and encrypted with a password before transfer.

## Requirements

- Go 1.21 or newer
- Both computers on the same local network

## Quick Start

### 1. Setup (do this on both computers)

```bash
git clone https://github.com/kikikian/transfer
cd transfer
go run main.go
```

Then open your browser to: **http://localhost:3030**

You should see a home page with two options: **Send** and **Receive**.

---

## How to Transfer Files

### Step 1: Find Your IP Address

First, determine the receiver's IP address so the sender knows where to send files.

**On Windows:**
```bash
ipconfig
```
Look for IPv4 Address (usually starts with `192.168.x.x` or `10.x.x.x`)

**On Linux/Mac:**
```bash
ip addr
```
Look for `inet` under your network interface.

### Step 2: Receiver - Start Listening

**On the receiving computer:**

1. Go to http://localhost:3030
2. Click **Receive**
3. Choose the folder where you want to save files
4. Enter a password (share this with the sender)
5. Click **Receive** — it will wait for the connection

### Step 3: Sender - Send Files

**On the sending computer:**

1. Go to http://localhost:3030
2. Click **Send**
3. Choose the folder you want to send
4. Enter the **same password** as the receiver
5. Enter the receiver's IP address in the format: `192.168.1.45:8080` (replace with actual IP)
6. Click **Send** — files will compress and transfer automatically

Watch the progress bar as files upload and transfer across the network.

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
	