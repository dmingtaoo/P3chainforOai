@echo off
if exist p3Chain.exe (
    del p3Chain.exe
    go build p3Chain.go
) else (
    go build p3Chain.go
)

if exist daemon.exe (
    del daemon.exe
    go build .\daemonfile\daemon.go
) else (
    go build .\daemonfile\daemon.go
)

if exist daemonClose.exe (
    del daemonClose.exe
    go build .\daemonfile\closeScript\daemonClose.go
) else (
    go build .\daemonfile\closeScript\daemonClose.go
)

if exist .\..\..\chain_code_example\example_didSpectrumTrade\example_didSpectrumTrade.exe (
    del .\..\..\chain_code_example\example_didSpectrumTrade\example_didSpectrumTrade.exe
    go build  .\..\..\chain_code_example\example_didSpectrumTrade\example_didSpectrumTrade.go
) else (
    go build .\..\..\chain_code_example\example_didSpectrumTrade\example_didSpectrumTrade.go
)







