return function(app)
    app.Log("Starting init.lua")
    app.BindEvents("app", "Ctrl-C", app.Quit)
    app.BindEvents("app", "Resize.Size", function(win, size) app.Resize(size.W, size.H) end)

    app.BindEvents("app", "Ctrl-X Ctrl-N", function() app.SwitchWindow() end)

    local function runCurrentChunk(win)
        local err = win.Buffer().RunCurrent()
        if err ~= nil then
            app.Logf("Lua error: %s", err)
        else
            win.MoveCursorToEnd()   
        end
    end

    app.BindEvents("luarepl", "Enter", function(win)
        local l = win.CursorLine()
        if win.Buffer().IsCurrentLast(l) then
            win.DeleteRune()
            runCurrentChunk(win)
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
