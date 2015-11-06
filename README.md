# zrpm

Zrpm is a utility that can check for and automatically download and install updated RPM packages in ROSA Linux or Mandriva. Dependencies are obtained and downloaded automatically, prompting the user for permission as necessary.

## Usage

  * `zrpm repo` - Display information about a repositories.
  * `zrpm search` - Search for a package by name.
  * `zrpm show` or `zrpm info` - Display detailed information about a package.
  * `zrpm install` - Install/upgrade packages.
  * `zrpm remove` - Remove packages.
  * `zrpm update` - Download lists of new/upgradable packages.
  * `zrpm upgrade` - Perform an upgrade, possibly installing and removing packages.
  * `zrpm download` - Download binary RPMs.
  * `zrpm source` - Download the source RPMs (SRPMs).
  * `zrpm files` - List files in package or which package has installed file.
  * `zrpm help` - Shows a list of commands or help for one command

## Installation

You don't need to build the project to use it - you can use any of our [pre-built binaries](https://github.com/SokoloffA/zrpm/releases). These are standalone binaries that can be unpacked and executed on your system. They can be unpacked in a location such as

## License

By utilizing this software, you agree to the terms of the included license. Zrpm is licensed under the MIT agreement. See [LICENSE](https://raw.githubusercontent.com/SokoloffA/zrpm/master/LICENSE) for the full license terms.