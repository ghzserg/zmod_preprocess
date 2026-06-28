To add these profiles to OrcaSlicer:
- You must first add the stock AD5X profiles in OrcaSlicer (you must add every
  nozzle size you wish to use).
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


Start and filament change gcodes are set up to support USE_TRASH_ON_PRINT=0
(nopoop mode) and USE_TRASH_ON_PRINT=2 (slicer-controlled poop mode). The
default settings are configured for nopoop mode.

Make sure to set your USE_TRASH_ON_PRINT setting back to your intended value
after using these profiles. You can disregard this if you will be using these
profiles for all of your prints (as your next print will set the intended one
at the start anyway).

In either case, purge amounts will be determined by your Flush Volumes setting
in the slicer.


-= Requirements to print in "Nopoop" mode =-
All of the following conditions must be met to use nopoop mode.

1. "Wipe tower type" (Printer settings -> Multimaterial) must be "Type 2"
2. "Purge in prime tower" (Printer settings -> Multimaterial) must be enabled
3. Your print sequence must be "By Layer" (not "By Object")
4. Don't forget to check your flush volumes.

If any of these conditions are not met, the profiles will use slicer-controlled
poop mode instead.

Options such as "Purge to infill", "Purge to this object", etc will always work
correctly when using "Nopoop" mode, provided that Nopoop mode itself is working
correctly.


-= Requirements to print in "Slicer-controlled poop" mode =-
Slicer-controlled poop mode will be activated if even ONE of the following
conditions are met:

1. "Wipe tower type" (Printer settings -> Multimaterial) is "Type 1"
2. "Purge in prime tower" (Printer settings -> Multimaterial) is disabled
3. Your print sequence is "By Object"

(And don't forget to check your flush volumes!)

The options like "Purge to infill" etc will only work correctly in slicer-
controlled poop mode, if the "Wipe tower type" setting is set to Type 1. They
will have incorrect behavior otherwise.


-= Recap on "Purge to infill", "Purge to this object", etc =-

Nopoop = YES, you can use these options
Slicer-controlled poop + Type 1 wipe tower = YES, you can use these options
Slicer-controlled poop + Type 2 wipe tower = NO, these options will misbehave
