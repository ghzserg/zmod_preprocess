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


Start and filament change gcodes are set up to support USE_TRASH_ON_PRINT=0 (nopoop mode) 
and USE_TRASH_ON_PRINT=2 (slicer-controlled poop mode). 

To switch to slicer-controlled poop mode, disable the "Purge in prime tower" setting in the printer
Multimaterial settings. If the plate type is set to "By object", slicer-controlled be automatically used. 

Make sure to set your USE_TRASH_ON_PRINT setting back to your intended value after using these profiles.
You can disregard this if you will be using these profiles for all of your prints.

** A note on "Purge to infill", "Purge to object", etc: **
If you want to use these options with slicer-controlled poop, you must change the prime tower type
to type 1. You must turn "Purge to prime tower" off *BEFORE* changing the type.
