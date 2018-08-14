# laptop-booter

Brings full-disk-encrypted laptop from shutdown state to fully working in the home network from outside

## Very use-case driven

This is meant for my use case and my personal setup where I have a bastion SSH node
that I use to reach out to an AMT, initramfs dropbear and full-SSH endpoints to do following things:

1. `activate` will power on via AMT, reach out to dropbear via SSH and send disk decryption password 
  and then wait for the SSH server running on the laptop as a package ("real SSH") 
  to be on as a proof that "full power on" process has been completed;
2. `shutdown` will power off via AMT in case no real SSH is available 
  (which means machine is only partially on or working incorrectly), 
  _or_ use shell command `shutdown -h` via "real SSH" to turn off the computer;
3. `status` which will provide feedback about current AMT state.

I expect and require the use of SSH agent since I use Yubikey and thus no "private key" file is available.

Feel free to use this application as a script for your own use case or allow for even more switches to customize further.

For example, it currently works only when outside of my home network, from within it usage of bastion node
and the tunnels can be completely removed.
 
## Example run params

```bash
laptop-booter \
  -username admin \
  -password $AMT_PASSWORD  \
  -bastionHost $HOSTNAME_BASTION -bastionPort 1761 \
  -amtHost $IP_OF_AMT_NETWORK_INTF \
  -dropbearHost $IP_WHEN_ON -dropbearPort 4748 \
  -diskUnlockPassword $DISK_PASSWORD \
  -realSSHHost $IP_WHEN_ON -realSSHPort 22 \
  -command status
```
