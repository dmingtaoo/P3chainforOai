rd /s /q keys
rd /s /q log
md action
md keys
md bootnodes
md log
md pubkeys
@REM type nul > action/actionlist.txt
@REM type nul > bootnodes/nodes.txt
call keygenerate.bat
call iniserver.bat
