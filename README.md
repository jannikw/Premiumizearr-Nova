# Premiumizearr-Nova
## Overview
More information and Guides coming soon

Continuation and Improvement of the Premiumizearr Arr* Bridge Download Client compatible with Sonarr and Radarr.

This project is based on code from [Jackdallas' Premiumizearr](https://github.com/jackdallas/premiumizearr). 
It aims to improve its function and fix bugs as the Original Repo has gone stale and does not respond to issues and pull requests.
The code has been reused with modifications to suit my own use case.

* Fixes EOF Datastream Error
* Drastically improved download speed
* Updated permission sets
* Much more to come

## Running
First Pre-Release v1.3.0-rc1 available.

Prerequisites:
* Install Docker
* Create Folder Structure for mounting
* Make sure all Folders and Files are owned and accessible for UID and GID 1000
* Create or choose the correct Docker Network
* You might need to run it as a User with UID GID 1000 too (i havent tested yet)
* Important: Do not use sudo

Docker Run:
```bash
docker run -d --name premiumizearr \
  --network=compose_default \
  -v /home/user/premiumize/data:/data \
  -v /home/user/premiumize/blackhole:/blackhole \
  -v /home/user/premiumize/downloads:/downloads \
  -v /home/user/premiumize/unzip:/unzip \
  -e PGID=$(id -g) \
  -e PUID=$(id -u) \
  -p 8182:8182 \
  --restart unless-stopped \
  ghcr.io/ensingerphilipp/premiumizearr-nova:1.3.0-rc1
```

## License

This project is licensed under the **GNU General Public License v3.0** - see the [LICENSE](./LICENSE) file for details.

### Original Code

This project reuses code from [Jackdallas' Premiumizearr](https://github.com/jackdallas/premiumizearr), which is licensed under the **GNU General Public License v3.0**.

### Modifications

The following changes have been made to the original code:
- See Commit History
  
All modifications to the original code are also licensed under the same license, i.e., **GNU GPL v3**.

## Installation

TBD

3. Follow any additional installation instructions specific to your project.


### Example:
