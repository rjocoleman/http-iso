# HTTP-ISO Server

HTTP-ISO Server is a project that provides a simple HTTP server to inspect, access files, and netboot from ISO images without the need to mount them. You can download it from the [GitHub releases](https://github.com/rjocoleman/http-iso/releases) page.

## Description

HTTP-ISO Server is written in Go and uses the `github.com/kdomanski/iso9660` library to read ISO9660 images and serve files over HTTP. It generates an HTML file index for the ISO allowing for the exploration of the file structure of the ISO image through a web browser and accessing individual files. In addition, it can serve iPXE boot scripts for netbooting if the necessary parameters are provided.

## Usage

Once downloaded from the GitHub releases page, you can run HTTP-ISO Server as follows:

```bash
./http-iso --iso path_to_iso --kernel /kernel --initrd /initrd1,initrd1 --initrd /initrd2,initrd2 --initrd /initrd3 --params "console=tty0"
```

This will start an HTTP server on port 8080, serving the content of the ISO image at `/path/to/your.iso`. If `--kernel` and `--initrd` parameters are provided and these files exist in the ISO image, a boot.ipxe script will be dynamically served at `/boot.ipxe` e.g.

```
#!ipxe

kernel http://your_ip:8080/kernel console=tty0
initrd http://your_ip:8080/initrd1 initrd1
initrd http://your_ip:8080/initrd2 initrd
initrd http://your_ip:8080/initrd3
boot
```

## Intent and Limitations

HTTP-ISO Server is intended to:

1. Quickly inspect the contents of ISO images through a web browser without having to mount the images, thanks to the automatically generated HTML file index.
2. Extract files from ISO images over HTTP.
3. Serve iPXE boot scripts to enable network booting from ISO images.

However, there are some limitations:

1. Due to the nature of the ISO9660 filesystem, file access can be slower than regular filesystems, particularly for large files. This is because the ISO9660 filesystem layout requires multiple read operations to retrieve a single file, and the HTTP server needs to read the entire file into memory to serve it.
2. Netbooting performance can be further limited by network conditions. If the ISO image and the netbooting client are not in the same local network, netbooting can be very slow or even fail due to network timeouts.
3. The server doesn't support modifying the ISO image or uploading files.

For best performance, it is recommended to use this server with a locally stored ISO image.

Important: When netbooting an image, the kernel parameters are crucial and must match the requirements of the bootable system in the ISO image. These parameters can often be found in the boot configuration files in the ISO image (for example, in `grub.cfg` or `isolinux.cfg` for Linux distributions).

For other iPXE-based booting needs, consider checking out [netboot.xyz](https://netboot.xyz/), a project that provides a wide range of network-bootable operating systems and utilities.
