## troubleshooting ##

#### Booting issue

#####Error message showing when the machine just booted.
```
/dev/nvme0n1p2: clean xxx/xxxx files, xxxx/xxxxx/blocks
```

This is because the graph card driver issues where the system was loaded in-correct graph card drivers. you can check relevant solutions here[[1](https://askubuntu.com/questions/1277842/ubuntu-20-04-stuck-at-dev-nvme0n1p5-clean-xxx-xxx-files-xxx-xxx-blocks-duri)].

To fix it, boot with recovery mode and update grub config with root permission, and remove `quiet splash` with `nomodeset` to enable boot with lower resolution to avoid driver issue.

```
gedit /etc/default/grub

- GRUB_CMDLINE_LINUX_DEFAULT="quiet splash"
+ GRUB_CMDLINE_LINUX_DEFAULT="nomodeset"

```

then update grub with `sudo update-grub`. then reboot would solve the issue.