return function(app)
    app.Log("Starting init.lua")
    app.BindEvents("app", "Ctrl-D", app.Quit)
    app.UnbindEvents("app", "Ctrl-C") -- This is the default key to quit
    app.BindEvents("app", "Resize.Size", function(win, size) app.Resize(size.W, size.H) end)

    app.BindEvents("app", "Ctrl-X Ctrl-N", function() app.SwitchWindow() end)

    local function runCurrentChunk(win)
        local more, err = win.Buffer().RunCurrent()
        if err ~= nil then
            app.Logf("Lua error: %s", err)
        else
            win.MoveCursorToEnd()   
        end
        return more
    end

    -- -- This requires pressing "Enter" twice to evaluate a chunk
    -- -- I like it but perhaps I'm the only one!
    -- app.BindEvents("luarepl", "Enter", function(win)
    --     local l = win.CursorLine()
    --     if win.Buffer().IsCurrentLast(l) then
    --         win.DeleteRune()
    --         runCurrentChunk(win)
    --     elseif win.SplitLine(true) ~= nil then
    --         -- Try copying
    --         local err = win.Buffer().CopyToCurrent(l)
    --         if err ~= nil then
    --             app.Logf("Lua error: %s", err)
    --         else
    --             win.MoveCursorToEnd()
    --         end
    --     end
    -- end)

    app.BindEvents("luarepl", "Enter", function(win)
        local l, c = win.CursorPos()
        if win.Buffer().IsEndOfCurrentInput(l, c) then
            if runCurrentChunk(win) then
                win.SplitLine(true)
            end
        elseif win.SplitLine(true) ~= nil then
            -- Try copying
            local err = win.Buffer().CopyToCurrent(l)
            if err ~= nil then
                app.Logf("Lua error: %s", err)
            else
                win.MoveCursorToEnd()
            end
        end
    end)

    app.BindEvents("luarepl", "Ctrl-X Ctrl-D", function(win)
        win.Buffer().ResetCurrent()
        win.MoveCursorToEnd()
    end)
end
