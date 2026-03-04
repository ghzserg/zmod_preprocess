To add these profiles to OrcaSlicer:
- You must first add the stock AD5X profiles in OrcaSlicer (you must add every nozzle size you wish to use).
- Then you can import them via File -> Import -> Import Configs.
- You can continue using your existing material and print process profiles.

Differences from stock AD5X profiles:
- Fan speed-up time set to 1.5s
- Fan kick-start time set to 0.5s
- zMod start / end / layer change gcode in place
- Start gcode modified to automatically set correct USE_TRASH_ON_PRINT mode
- Filament change gcode added
- Purge in prime tower enabled
- Filament load time set to 66s, unload time set to 0
- Z-hop type set to normal

Start and filament change gcodes are set up to support both USE_TRASH_ON_PRINT=0 (nopoop mode) and
USE_TRASH_ON_PRINT=2 (slicer-controlled poop mode). To switch to slicer-controlled poop mode, disable
the "Purge in prime tower" setting.

Make sure to set your USE_TRASH_ON_PRINT setting back to your intended value after using these profiles.
You can disregard this if you will be using these profiles for all of your prints.

** A note on "Purge to infill", "Purge to object", etc: **
These settings do not work correctly in poop mode (ie: Purge to prime tower disabled) due to a bug in
OrcaSlicer. Although they will be "purged" to, the full amount of poop will also be produced. You can
control poop volumes via Flush Volumes when using poop mode with these profiles, but you can only use
the purge redirection options when printing in nopoop mode.

As mentioned, this is due to a bug in OrcaSlicer itself, and is not something the profiles or zMod can
fix. If OrcaSlicer fixes this bug in the future, you will be able to use those options without any new
version of zMod or these profiles being required.