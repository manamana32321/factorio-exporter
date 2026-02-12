storage.bridge_events=storage.bridge_events or {}
storage.bridge_push=function(e)table.insert(storage.bridge_events,e)if #storage.bridge_events>1000 then table.remove(storage.bridge_events,1)end end
rcon.print("ok")
