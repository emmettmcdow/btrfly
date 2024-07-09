# btrfly - The HTTP Time Machine for Reproducible Builds
> **NOTE**: This software is currently a WIP. It is nowhere close to stable or usable.

This is a tool allows users to record a session of http traffic, then playback
that session at a later time.

This was built in retaliation to the prevailing artifact storage solutions. i.e. Artifactory
and Nexus. I find them annoying and cumbersome (I'm lazy). 

But build reproducibility is important, apparently.

## Example
Here's a simple example of one way you could use btrfly:
```bash
# This configures the build machine to utilize the btrfly service
btrfly start record --tag=example1

# ... do your build (including downloading stuff)
pip install requests
curl -LO https://download.rockylinux.org/pub/rocky/9/isos/x86_64/Rocky-9.4-x86_64-minimal.iso
go get github.com/syncthing/syncthing

btrfly stop record
```
Now, some amount of time later. Lets say, 1000 years later. Those servers for sure do not still exist.
If they do, that would be insane. By some miracle however, you still have access to the btrfly service,
and access to a machine capable of running ancient programs.
```bash
# This configures the build machine to utilize the btrfly service
btrfly start playback --tag=example1

# ... do your build (including downloading stuff)
pip install requests
curl -LO https://download.rockylinux.org/pub/rocky/9/isos/x86_64/Rocky-9.4-x86_64-minimal.iso
go get github.com/syncthing/syncthing

btrfly stop playback
```
This second example will pull artifacts from the artifacts it recorded the first time around. No 
need to pull them again. As long as the code you ran pulls the same URLs, btrfly will give the correct
data.

## How does it work?
btrfly is made up of three parts:

## btrfly CLI
- This configures the client/host to utilize the btrfly DNS server. 
- This also is used to authenticate the client against the server.
- This also is used to specify the `tag` and `mode` of the build

The tag can be though of like a docker tag. It is simply a reference to an underlying artifact.
The mode specifies whether btrfly should be behaving in `record` or `playback` mode.

## btrfly DNS Server
Every domain points to the btrfly proxy IP.

## btrfly Proxy
This is where the magic happens. This is just a proxy that injects its own data when necessary.
If we are recording, we pick up the body of every HTTP request and save it. If we are playing
back, we use our saved recordings. Which recorded HTTP bodies to use are determined by
which `tag` is currently active. Easy peasy. 
