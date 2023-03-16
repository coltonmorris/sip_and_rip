# What is this
A sip server that responds to calls by sending an audio file.

## Running
1. Make sure to add a mono ulaw encoded wav file named `ulaw-test.wav`
2. If on apple silicon:
```
GOARCH=amd64 go build . && ./sip_and_rip
```
3. Now dial the server on your sip client. Our domain is `127.0.0.1:5061`.
