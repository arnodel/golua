-- tags: windows

-- Test os.execute() with no args (shell available check)
print(os.execute())
--> =true

-- Test successful command (produces no output)
local ok, exitType, code = os.execute("echo. >nul")
print(ok, exitType, code)
--> =true	exit	0

-- Test command with non-zero exit code
ok, exitType, code = os.execute("exit /b 42")
print(ok, exitType, code)
--> =nil	exit	42

-- Test command with exit 0
ok, exitType, code = os.execute("exit /b 0")
print(ok, exitType, code)
--> =true	exit	0

-- Note: Windows doesn't have Unix signals, so no signal tests here

-- Test command that doesn't exist (shell runs but command fails)
ok, exitType, code = os.execute("nonexistent_command_12345 2>nul")
print(ok, exitType, code)
--> =nil	exit	1
