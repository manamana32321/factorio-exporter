local p=storage.bridge_push
script.on_event(defines.events.on_entity_died,function(e)if e.entity and e.entity.type=="unit-spawner"then p({type="spawner_destroyed",name=e.entity.name,tick=e.tick})end end)
script.on_event(defines.events.on_surface_created,function(e)local s=game.get_surface(e.surface_index)p({type="surface_created",name=s and s.name or"unknown",tick=e.tick})end)
script.on_event(defines.events.on_chart_tag_added,function(e)p({type="tag_added",text=e.tag.text or"",tick=e.tick})end)
rcon.print("ok")
