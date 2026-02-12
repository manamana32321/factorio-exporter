local p=bridge_push
script.on_event(defines.events.on_research_started,function(e)p({type="research_started",name=e.research.name,tick=e.tick})end)
script.on_event(defines.events.on_research_cancelled,function(e)p({type="research_cancelled",name=e.research.name,tick=e.tick})end)
script.on_event(defines.events.on_player_died,function(e)local pl=game.get_player(e.player_index)p({type="player_died",player=pl.name,cause=e.cause and e.cause.name or"unknown",tick=e.tick})end)
script.on_event(defines.events.on_player_respawned,function(e)p({type="player_respawned",player=game.get_player(e.player_index).name,tick=e.tick})end)
script.on_event(defines.events.on_player_changed_surface,function(e)local pl=game.get_player(e.player_index)p({type="player_changed_surface",player=pl.name,surface=pl.surface.name,tick=e.tick})end)
rcon.print("ok")
