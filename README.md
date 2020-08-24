# The One-Button Internet Telegraph

The easiest way to install the internet telegraph client is to use our pre-built SD card image: just download it from the [releases page](https://github.com/TheAutodidacts/InternetTelegraph/releases) and follow the installation instructions in the build tutorial.

But some of you may way want to tinker with the code, or have already have Raspbian installed and configured, and want to run the telegraph from within your existing installation. If that sounds like you, read on!

### Building the client from source

Install Golang for your platform. On Linux, you can do that with:

```
sudo apt install golang
```

Set the relevant environment variables with:

```
export GOOS=linux GOARCH=arm
```

`go get` the neccessary dependencies:

```
go get -u github.com/nsf/termbox-go
go get -u github.com/stianeikeland/go-rpio
go get -u golang.org/x/net/websocket
```

And build with:

```
go build -o internet-telegraph client.go
```

### Installing the telegraph software

First, install Raspbian by following Raspberry Pi’s [official installation instructions](https://www.raspberrypi.org/documentation/installation/).

There are two ways to go from here: you can take the SD card out of your Pi and add the necessary files manually, or you can do the entire thing over SSH.

**To install it on the SD card directly:**

1. Take the SD card out of your Pi and plug it into your SD card reader
- [Download the telegraph code](https://github.com/TheAutodidacts/InternetTelegraph/archive/master.zip) from GitHub.
- Build the telegraph client as described above, or download the prebuilt binary from the ["releases" page](https://github.com/TheAutodidacts/InternetTelegraph/releases/latest).
- Drag the internet telegraph binary (`internet-telegraph`) and the internet telegraph configuration file (`config.json`) into the root directory of your Pi.
- Drag `rc.local` into the `/etc` directory, and replace the rc.local that is already there. (If you’ve customized your rc.local for other reasons, copy the relevant portions into your rc.local rather than overwriting it.)
- Set up your Pi to connect to your wifi network by following the [official instructions on raspberrypi.org](https://www.raspberrypi.org/documentation/configuration/wireless/wireless-cli.md)
- Eject your SD card, pop it back into your Pi, and boot it up!

**To install the internet telegraph client over SSH:**

1. Drop a file called `ssh` (such as the one in this repository) into the `boot` partition of your SD card to enable SSH access.
- Boot up your Pi and connect it to the internet
- SSH into your Pi. You can use [nmap](http://nmap.org) (`nmap 192.168.1.0/24`) to find your Pi’s IP address, or try `ssh pi@raspberrypi.local`.
- Type in your Pi’s password (which you have hopefully changed from the default, "raspberry")
- [Download the telegraph code](https://github.com/TheAutodidacts/InternetTelegraph/archive/master.zip) from GitHub.
- Build the telegraph client for ARM, as described above, or download the prebuilt binary from the ["releases" page](https://github.com/TheAutodidacts/InternetTelegraph/releases/latest) and stick it in the same directory as the sourcecode.
- Copy the files to your Pi over SSH with the following three commands:

```
cd ~/Downloads/internet-telegraph-master/

scp internet-telegraph config.json pi@raspberrypi.local:/

scp rc.local pi@raspberrypi.local:/etc
```

 (replacing `~/Downloads/internet-telegraph-master` with the path to your local copy of the internet telegraph code).
- Test it manually with `./internet-telegraph`
- Reboot your Pi with `sudo reboot`
