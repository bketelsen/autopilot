# Autopilot

Autopilot is a cli tool and webserver that helps you provision an Ubuntu 24.04 VM that is ready for MDE/Intune enrollment.

## Usage

Autopilot is intended to be used on a system that is reachable from your target device/VM. This could be on the same host as your hypervisor, or one on the same network.

To start, run the `autopilot` command from a terminal/shell.

Autopilot will prompt you for the details required to provision the VM, print out the URL of the autoinstall that is generated, and start a webserver.

From your target device/hypervisor, start an installation of Ubuntu 24.04 Desktop. When asked, choose "Automated Install", then enter the `Autoinstall URL` that is printed by Autopilot.

The installation will reboot when complete. In the target, enter your encryption phrase to decrypt the disk, then login.  To install the Intune components run `/root/intune.sh` as the `root` user from a command prompt.

## Assumptions

Autopilot assumes the `openssl` command is available to hash the user password.

## TODO

- Add common-sense packages that everyone will need
- Figure out hyper-v / TPM Full disk encryption
- replicate `mkpasswd` or `openssl password` in code to drop the dependency and make it work in windows
