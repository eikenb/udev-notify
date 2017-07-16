Udev Device Connection Nofity/Trigger
-------------------------------------

This is a tool that watches for connecting devices via udev, it looks for ones
that match based on a property and runs a configured command. I use this with
my laptop to configure my trackball and to set my USB DAC as the default-sink
for pulseaudio when it is plugged in. I run it as part of my user session as a
systemd user service.

The TOML formatted config files can either go under the XDG_CONFIG_HOME
directory, by default this is...

  ~/.config/udev-notify/config.toml

Or you can tell it the path to the file via the environment variable.

  UdevNotifyConfig

See the ./example-config.toml for more on how to set it up.


Command line options
--------------------
If call with '-list' it will list all your HID devices and their properties. To
make it easier to configure new entries.

Copyright
---------

[CC0](http://creativecommons.org/publicdomain/zero/1.0/); To the extent
possible under law, John Eikenberry has waived all copyright and related or
neighboring rights to this work. This work is published from: United States.
