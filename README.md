USB HID Connection Monitor/Trigger
----------------------------------

This is a tool that watches for connecting USB HID devices, looks for ones that
match based on some property and runs a configured command. I use this to
configure my trackball and to set my USB DAC as the default-sink for pulseaudio
when it is plugged in. I run it as part of my user session as a systemd user
service.

Right now the property/value/command configuration is just hard coded at the
top. I might add a config file if I feel like it or someone asks.

If call with '-list' it will list all your HID devices and their properties. To
make it easier to configure new entries.

Copyright
---------

[CC0](http://creativecommons.org/publicdomain/zero/1.0/); To the extent
possible under law, John Eikenberry has waived all copyright and related or
neighboring rights to this work. This work is published from: United States.
