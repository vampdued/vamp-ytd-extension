Set objShell = CreateObject("WScript.Shell")
' Run the bridge executable invisibly (0 means hidden window)
objShell.Run "d:\ytd\bridge.exe", 0, False
