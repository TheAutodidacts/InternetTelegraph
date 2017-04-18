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

And build with:

```
go build -v internet-telegraph.go
```

### Installing the telegraph software

First, install Raspbian by following Raspberry Pi’s [official installation instructions](https://www.raspberrypi.org/documentation/installation/).

There are two ways to go from here: you can take the SD card out of your Pi and add the necessary files manually, or you can do the entire thing over SSH.

**To install it on the SD card directly:**

1. Take the SD card out of your Pi and plug it into your SD card reader
2. [Download the telegraph code](https://github.com/TheAutodidacts/InternetTelegraph/archive/master.zip) from GitHub.
3. Drag the internet telegraph binary (`internet-telegraph`) and the internet telegraph configuration file (`config.json`) into the root directory of your Pi.
4. Drag `rc.local` into the `/etc` directory, and replace the rc.local that is already there. (If you’ve customized your rc.local for other reasons, copy the relevant portions into your rc.local rather than overwriting it.)
5. Set up your Pi to connect to your wifi network by following the [official instructions on raspberrypi.org](https://www.raspberrypi.org/documentation/configuration/wireless/wireless-cli.md)
6. Eject your SD card, pop it back into your Pi, and boot it up!

**To install the internet telegraph client over SSH:**

1. Install [nmap](http://nmap.org) if you don’t have it already, and then scan your network with the command `nmap 192.168.1.0/24`.
2. Boot up your Pi and connect it to the internet
3. Run nmap again, and notice which IP address is new: that should be your Pi
4. SSH into your Pi with `ssh pi@192.168.1.123`, replacing 192.168.1.123 with the IP address you find with nmap. (Or try `ssh pi@raspberrypi.local`)
5. Type in your Pi’s password (which you have hopefully changed from the default, "raspberry")
6. Download the internet telegraph code from GitHub
7. Copy over the Telegraph files with `scp ~/Downloads/internet-telegraph/internet-telegraph pi@192.168.1.123:/ && scp ~/Downloads/internet-telegraph/config.json pi@192.168.1.123:/ && ~/Downloads/internet-telegraph/rc.local pi@192.168.1.123:/etc` (replacing `192.168.1.123` with your Pi’s IP address, and `~/Downloads/internet-telegraph` with the path to your local copy of the internet telegraph code). Test it manually with `./internet-telegraph`, and then reboot your Pi.
