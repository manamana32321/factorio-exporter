local p=storage.bridge_push
script.on_event(defines.events.on_player_promoted,function(e)p({type="player_promoted",player=game.get_player(e.player_index).name,tick=e.tick})end)
script.on_event(defines.events.on_player_demoted,function(e)p({type="player_demoted",player=game.get_player(e.player_index).name,tick=e.tick})end)
script.on_event(defines.events.on_rocket_launch_ordered,function(e)p({type="rocket_launch_ordered",tick=e.tick})end)
script.on_event(defines.events.on_space_platform_changed_state,function(e)local pl=e.platform p({type="platform_state_changed",name=pl.name,state=tostring(pl.state),tick=e.tick})end)
script.on_event(defines.events.on_cargo_pod_finished_ascending,function(e)p({type="cargo_ascended",tick=e.tick})end)
script.on_event(defines.events.on_cargo_pod_finished_descending,function(e)p({type="cargo_descended",tick=e.tick})end)
rcon.print("ok")
