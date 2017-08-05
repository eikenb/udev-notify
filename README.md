Udev Device Connection Nofity/Trigger
-------------------------------------

A tool to watch for device Udev events, matching on a property and running a
configured command. Designed to run as part of a user session, add it to your
appropriate place for your
[window-manager or desktop](https://wiki.archlinux.org/index.php/Autostarting).

[![Build Status](https://travis-ci.org/eikenb/udev-notify.svg?branch=master)](https://travis-ci.org/eikenb/udev-notify)

Getting Started
---------------

Install the software. Via go get..

    go get github.com/eikenb/udev-notify

Or download a release.

Say you want to run some xinput commands to configure your mouse when you plug
it in. First you need to create a config rule for it, for which you need some
information. To get this run udev-notify in watch mode and plug in your mouse.

    udev-notify -w all
    (plug in mouse)

It will spit out a list of properties for that device event. Note the
SUBSYSTEM, ACTION and another property that would be unique among that type of
subsystem, like the NAME or ID_MODEL. You write up the commands in a script and
put it all in your config file.

An entry would look something like this..

    [[Rules]]
    Subsystem = "input"
    Action    = "add"
    PropName  = "ID_MODEL"
    PropValue = "Kensington_Slimblade_Trackball"
    Command   = "xinput-slimblade"

It searches for a TOML formatted config file passed on the command line or in
`$XDG_CONFIG_HOME/udev-notify/config.toml`.

See the ./example-config.toml for the config file structure.


NOTE: By default XDG_CONFIG_HOME is set to ~/.config on most Linux systems.


Copyright
---------

[CC0](http://creativecommons.org/publicdomain/zero/1.0/); To the extent
possible under law, John Eikenberry has waived all copyright and related or
neighboring rights to this work. This work is published from: United States.
