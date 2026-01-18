-- Consolidated test file for attributes using load()

global print, load, io

local function test(name, code)
    print(name)
    local chunk, err = load(code, name)
    if chunk then
        print("ok")
    else
        print(err)
    end
end

--
-- Failing tests (Correct behavior is to fail compilation)
--

test("local prefix x fail", "local <const> x, y = 1, 2; x = 3")
--> =local prefix x fail
--> ~^local prefix x fail:1:.*: attempt to reassign constant variable 'x'

test("local prefix y fail", "local <const> x, y = 1, 2; y = 3")
--> =local prefix y fail
--> ~^local prefix y fail:1:.*: attempt to reassign constant variable 'y'

test("local postfix fail", "local x<const>, y = 1, 2; x = 3")
--> =local postfix fail
--> ~^local postfix fail:1:.*: attempt to reassign constant variable 'x'

test("global prefix x fail", "global <const> gx, gy = 1, 2; gx = 3")
--> =global prefix x fail
--> ~^global prefix x fail:1:.*: attempt to assign to const global variable 'gx'

test("global prefix y fail", "global <const> gx, gy = 1, 2; gy = 3")
--> =global prefix y fail
--> ~^global prefix y fail:1:.*: attempt to assign to const global variable 'gy'

test("global postfix fail", "global gx<const>, gy = 1, 2; gx = 3")
--> =global postfix fail
--> ~^global postfix fail:1:.*: attempt to assign to const global variable 'gx'

test("global close prefix fail", "global <close> gx")
--> =global close prefix fail
--> ~^global close prefix fail:1:.*: only <const> is allowed for global prefix

test("global close postfix fail", "global gx <close>")
--> =global close postfix fail
--> ~^global close postfix fail:1:.*: only <const> is allowed on global declarations

--
-- Succeeding tests (Correct behavior is to compile successfully)
--

test("valid local postfix", "local x<const>, y = 1, 2; y = 3")
--> =valid local postfix
--> =ok

test("valid global postfix", "global gx<const>, gy = 1, 2; gy = 3")
--> =valid global postfix
--> =ok

--
-- Test for <close> implying <const>
--
test("close implies const", "local <const> x, y<close> = 1, io.stdin; y = io.stdout")
--> =close implies const
--> ~^close implies const:1:.*: attempt to reassign constant variable 'y'

test("override prefix with postfix 2", "global <const> gx, gy<const> = 1, 2")
--> =override prefix with postfix 2
--> =ok