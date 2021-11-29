tarserv
=======

A collection of tools that allow serving large datasets from local filesystem snapshots.

It is meant for serving big amounts of data to shell scripted receivers.

Process:
 
  1. Take snapshot of directory containing data "/var/datadir/.snapshot12345".
  2. Create build index of the snapshot and store it into index directory:\
     `$ createindex /var/index/snapshot12345.taridx /var/datadir/.snapshot12345`
  3. Serve snapshots via HTTP: `$ tarserv -l 127.0.0.1:8080 -i /var/index/`
  4. Download snapshots via HTTP: `$ curl http://127.0.0.1:8080/snapshot12345/data.tar`

Features:

  - The tar files are created on the fly from the index file and filesystem snapshot/directory.
  - Downloading is resilient, it allows range requests and so can be efficiently restarted.
  - Furthermore the tar files produced contain a file ".version" that contains the source snapshot name.
  - If given the "lastfile" GET parameter, the served snapshot will start with the file named by the parameter.
  - All contained paths are normalized to current directory "./".
