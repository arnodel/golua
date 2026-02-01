-- tags: !windows

-- Test os.execute() with no args (shell available check)
print(os.execute())
--> =true

-- Test successful command (use true which produces no output)
local ok, exitType, code = os.execute("true")
print(ok, exitType, code)
--> =true	exit	0

-- Test command with non-zero exit code
ok, exitType, code = os.execute("exit 42")
print(ok, exitType, code)
--> =nil	exit	42

-- Test command with exit 0
ok, exitType, code = os.execute("exit 0")
print(ok, exitType, code)
--> =true	exit	0

-- Test signal termination (SIGKILL = 9)
ok, exitType, code = os.execute("sh -c 'kill -9 $$'")
print(ok, exitType, code)
--> =nil	signal	9

-- Test signal termination (SIGTERM = 15)
ok, exitType, code = os.execute("sh -c 'kill -15 $$'")
print(ok, exitType, code)
--> =nil	signal	15

-- Test command that doesn't exist (shell runs but command fails with 127)
ok, exitType, code = os.execute("nonexistent_command_12345 2>/dev/null")
print(ok, exitType, code)
--> =nil	exit	127
