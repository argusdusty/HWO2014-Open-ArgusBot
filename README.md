## HWO 2014 Argusdusty Bot

This is a copy of the bot that set most of the records.
After building it with ./build, run it with ./bot testserver.helloworldopen.com 8091 imola > race.dat
Or run a multi-car race with ./bot hakkinen.helloworldopen.com 8091 suzuka 4 foo argus > race.dat

If you're worried about performance, to get it to run in the real-time limit, you can lower the resolution on
DoThrottle (from 1<<5, 1<<3 to 1<<3, 1<<2, or something), and switch from DoTurbo2 (really slow), back to DoTurbo.

I'll try to add an explaination of all the physics and algorithms soon.