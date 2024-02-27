#! /bin/sh

if [ ! -f "p3Chain" ]; then
	go build p3Chain.go
	else
	rm -f p3Chain
	go build p3Chain.go
fi

if [ ! -f "daemon" ]; then
	go build ./daemonfile/daemon.go
	else
	rm -f daemon
	go build ./daemonfile/daemon.go
fi

if [ ! -f "daemonClose" ]; then
	go build ./daemonfile/closeScript/daemonClose.go
	else
	rm -f daemonClose
	go build ./daemonfile/closeScript/daemonClose.go
fi

if [ ! -f "example_didSpectrumTrade" ]; then
	go build ./../../chain_code_example/example_didSpectrumTrade/example_didSpectrumTrade.go
	else
	rm -f example_stamp
	go build ./../../chain_code_example/example_didSpectrumTrade/example_didSpectrumTrade.go
fi


