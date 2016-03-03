## Lists the set of meta-data thats kept for a dvol, this list
## is a start and extensions can be done any time. Add an entry
## for the index for the new key and add the key string in the
## array.
##
## Note: To add a new property add to the end of the DVOL_* list
## and add the key string to the end of the kv_string list.

# Key IDs, the kv store is indexed using these
DVOL_VOLUME = 0
DVOL_VM_ID = 1
DVOL_DAEMON_ID = 2
DVOL_STATUS = 3
DVOL_VOLOPTS = 4
DVOL_CBRC_ENABLED = 5
DVOL_IOFILTERS = 6
DVOL_CONTROLLER = 7
DVOL_SLOT = 8

# Key names
kv_strings = ["dvolVolume", "dvolVMID", "dvolDaemonID", "dvolStatus", "dvolVolOpts", "dvolCBRCEnabled", "dvolIOFilters", "controller", "slot"]
