# laptop-booter

Brings full-disk-encrypted laptop from shutdown state to fully working in the home network from outside

## Very use-case driven

This is meant for my use case and my personal setup where I have a bastion SSH node
that I use to reach out to an AMT, initramfs dropbear and full-SSH endpoints to do following things:

1. `activate` will power on via AMT, reach out to dropbear and send disk decryption password 
  and then wait for SSH server to be on as a proof that full power on has been completed
2. `shutdown` will power off via AMT in case no real SSH is available 
  (which means machine is only partially on or working incorrectly), 
  or use correct `shutdown -h` to turn off the server
3. `status` which will provide feedback about current AMT state

Feel free to use this application as a script for your own use case or allow for switches.

For example, it currently works only when outside of my home network, from within it usage of bastion node
and the tunnels can be completely removed.
 