NOTE: These profiles have not been tested to the same extent as the OrcaSlicer profiles. Use at your own risk!

To add these profiles to Bambu Studio:
- You must first add the Creality K1 0.4mm profile to Bambu Studio.
- Then you can import them via File -> Import -> Import Configs.
- The default print process profiles are suitable for a 0.4mm nozzle only; you will need to create your
  own print process profiles for other nozzle sizes.

These are made to replicate the stock OrcaSlicer (not Orca-Flashforge) AD5X profiles as closely as possible,
outside of a few intentional differences:
- zMod start / end / layer change gcode in place
- Start gcode modified to automatically set USE_TRASH_ON_PRINT to 2
- Filament change gcode added
- Filament load time set to 66s, unload time set to 0
- Z-hop type set to normal

USE_TRASH_ON_PRINT = 0 (nopoop mode) cannot be used with Bambu Studio. This profile only has support for
USE_TRASH_ON_PRINT = 2.

Make sure to set your USE_TRASH_ON_PRINT setting back to your intended value after using these profiles.
You can disregard this if you will be using these profiles for all of your prints.

** A note on "Purge to infill", "Purge to object", etc: **
These options can be used with these profiles and will work correctly. Bambu Studio does NOT have the same
bug as OrcaSlicer that prevents them being used outside of nopoop mode.