/*
The package udev-notify is a tool to watch for device Udev events, matching on
a property and running a configured command. Designed to run as part of a user
session on a Linux system.

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

It searches for a TOML formatted config file passed on the command line or in..

	$XDG_CONFIG_HOME/udev-notify/config.toml

See the example-config.toml for the config file structure.

NOTE: By default XDG_CONFIG_HOME is set to ~/.config on most Linux systems.

NOTE: Udev can get triggered sometimes at odd times (docker seems to trigger
some events). So it is best to try to make your commands idempotent.
*/
package main
